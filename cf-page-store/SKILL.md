---
name: cf-page-store
description: Run one private Cloudflare store for phone-page pages and their latest submissions.
---

# Cloudflare page store

Use `cf-page-store publish <file>` and `cf-page-store answers <id>` as phone-page's store command. The Worker requires a valid Cloudflare Access token on every route, including assets.
Answered pages are deleted three days after their last submission.

## Local file

`~/.agents/local/cf-page-store.md` holds every machine value. Read it when it exists. Its complete format is six lines in this order:

```text
Store address: <workers.dev URL>
Cloudflare account id: <account id>
Team domain: <full HTTPS team domain>
Audience tag: <Access application audience tag>
D1 database id: <database id>
Deploy config: <absolute path to the rendered Wrangler config>
```

The OS keyring holds three generic-password entries under service `cf-page-store`: accounts `api-token`, `service-token-id` and `service-token-secret`. No token belongs in the local file.

## Install

Run these steps in order. Cloudflare commands run once during install; never run them in continuous integration.

1. Ask the user to create a Zero Trust organization on Cloudflare's free plan in the dashboard. Cloudflare asks for a payment method, but the free plan charges nothing. Record the full team domain. Open Workers & Pages once in the dashboard so Cloudflare registers the `workers.dev` subdomain before deployment.

2. Ask the user to create one API token with Workers Admin, D1 Write, Access: Apps and Policies Write, Access: Service Tokens Write. The first deploy creates a new Worker and needs Workers Admin; Workers Editor updates only existing Workers. Store it without printing it:

   ```sh
   # Linux
   secret-tool store --label='cf-page-store Cloudflare API token' service cf-page-store username api-token

   # macOS
   security add-generic-password -U -s cf-page-store -a api-token -w
   ```

3. From this directory, install the pinned packages, create D1, apply the schema, render the deploy config next to the future local file, download the phone-page assets, and deploy. Set each placeholder to the value from the dashboard or command output. The first deploy uses an audience that matches no Access application because Access does not exist yet.

   ```sh
   npm ci
   export ACCOUNT_ID='<cloudflare-account-id>'
   export DATABASE_NAME='<database-name>'
   export WORKER_NAME='<worker-name>'
   export TEAM_DOMAIN='<full-https-team-domain>'
   export AUDIENCE_TAG='pending-access-application'
   export DEPLOY_CONFIG="$HOME/.agents/local/cf-page-store.wrangler.jsonc"

   case "$(uname -s)" in
     Linux) export CLOUDFLARE_API_TOKEN="$(secret-tool lookup service cf-page-store username api-token)" ;;
     Darwin) export CLOUDFLARE_API_TOKEN="$(security find-generic-password -s cf-page-store -a api-token -w)" ;;
   esac
   export CLOUDFLARE_ACCOUNT_ID="$ACCOUNT_ID"

   npx wrangler d1 create "$DATABASE_NAME"
   export D1_DATABASE_ID='<database-id-from-the-create-command>'
   npx wrangler d1 execute "$DATABASE_NAME" --remote --file migrations/0001_schema.sql
   mkdir -p "$(dirname "$DEPLOY_CONFIG")"
   jq \
     --arg name "$WORKER_NAME" \
     --arg entrypoint "$PWD/src/index.ts" \
     --arg account "$ACCOUNT_ID" \
     --arg team "$TEAM_DOMAIN" \
     --arg audience "$AUDIENCE_TAG" \
     --arg databaseName "$DATABASE_NAME" \
     --arg databaseId "$D1_DATABASE_ID" \
     --arg assets "$PWD/assets" \
     '.name = $name | .main = $entrypoint | .account_id = $account |
      .vars.ACCESS_TEAM_DOMAIN = $team | .vars.ACCESS_AUDIENCE = $audience |
      .d1_databases[0].database_name = $databaseName |
      .d1_databases[0].database_id = $databaseId |
      .assets.directory = $assets' \
     wrangler.template.jsonc >"$DEPLOY_CONFIG"
   phone-page assets "$PWD/assets"
   npx wrangler deploy --config "$DEPLOY_CONFIG"
   export STORE_ADDRESS='<deployed-workers.dev-url>'
   ```

4. Create the service token, retain its rule id for step 5, and store only its client id and secret in the keyring:

   ```sh
   service_token=$(printf 'Authorization: Bearer %s\n' "$CLOUDFLARE_API_TOKEN" |
     curl -fsS "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/access/service_tokens" \
       --request POST \
       --header @- \
       --json '{"name":"cf-page-store","duration":"forever"}')
   export SERVICE_TOKEN_RULE_ID="$(jq -er '.result.id' <<<"$service_token")"
   service_token_id="$(jq -er '.result.client_id' <<<"$service_token")"
   service_token_secret="$(jq -er '.result.client_secret' <<<"$service_token")"
   case "$(uname -s)" in
     Linux)
       printf '%s' "$service_token_id" | secret-tool store --label='cf-page-store service token id' service cf-page-store username service-token-id
       printf '%s' "$service_token_secret" | secret-tool store --label='cf-page-store service token secret' service cf-page-store username service-token-secret
       ;;
     Darwin)
       printf 'add-generic-password -U -s cf-page-store -a service-token-id -w %q\n' "$service_token_id" | security -i
       printf 'add-generic-password -U -s cf-page-store -a service-token-secret -w %q\n' "$service_token_secret" | security -i
       ;;
   esac
   unset service_token service_token_id service_token_secret
   ```

5. Create the Access application and its two policies. Replace the email placeholder with the user's Cloudflare account email. A Sign in with Apple account uses its Apple private relay address. Then render the returned audience into the deploy config and deploy again.

   ```sh
   export USER_EMAIL='<cloudflare-account-email>'
   application=$(printf 'Authorization: Bearer %s\n' "$CLOUDFLARE_API_TOKEN" |
     curl -fsS "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/access/apps" \
       --request POST \
       --header @- \
       --json "$(jq -nc --arg domain "${STORE_ADDRESS#https://}" \
         '{name:"cf-page-store",domain:$domain,type:"self_hosted",session_duration:"24h"}')")
   export ACCESS_APPLICATION_ID="$(jq -er '.result.id' <<<"$application")"
   export AUDIENCE_TAG="$(jq -er '.result.aud' <<<"$application")"
   unset application

   printf 'Authorization: Bearer %s\n' "$CLOUDFLARE_API_TOKEN" |
     curl -fsS "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/access/apps/$ACCESS_APPLICATION_ID/policies" \
       --request POST \
       --header @- \
       --json "$(jq -nc --arg email "$USER_EMAIL" \
         '{name:"allow user",decision:"allow",precedence:1,include:[{email:{email:$email}}]}')" >/dev/null
   printf 'Authorization: Bearer %s\n' "$CLOUDFLARE_API_TOKEN" |
     curl -fsS "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/access/apps/$ACCESS_APPLICATION_ID/policies" \
       --request POST \
       --header @- \
       --json "$(jq -nc --arg token "$SERVICE_TOKEN_RULE_ID" \
         '{name:"allow store command",decision:"non_identity",precedence:2,include:[{service_token:{token_id:$token}}]}')" >/dev/null

   jq --arg audience "$AUDIENCE_TAG" '.vars.ACCESS_AUDIENCE = $audience' \
     "$DEPLOY_CONFIG" >"$DEPLOY_CONFIG.new"
   mv "$DEPLOY_CONFIG.new" "$DEPLOY_CONFIG"
   npx wrangler deploy --config "$DEPLOY_CONFIG"
   ```

6. Link the command into `PATH`, write this skill's local file, and set phone-page to use it:

   ```sh
   mkdir -p "$HOME/.local/bin" "$HOME/.agents/local"
   ln -s "$PWD/cf-page-store" "$HOME/.local/bin/cf-page-store"
   printf '%s\n' \
     "Store address: $STORE_ADDRESS" \
     "Cloudflare account id: $ACCOUNT_ID" \
     "Team domain: $TEAM_DOMAIN" \
     "Audience tag: $AUDIENCE_TAG" \
     "D1 database id: $D1_DATABASE_ID" \
     "Deploy config: $DEPLOY_CONFIG" \
     >"$HOME/.agents/local/cf-page-store.md"
   printf '%s\n' 'cf-page-store' >"$HOME/.agents/local/phone-page.md"
   unset CLOUDFLARE_API_TOKEN
   ```

7. Confirm that the store address without a login is turned away. Publish a phone-page question, open its link and complete it, then confirm `cf-page-store answers <id>` prints that submission. Before a submission, `answers` must print nothing and exit 3.

## Commands

- `cf-page-store publish <file>` sends one HTML file and prints exactly the Worker's one-line JSON object containing `id` and `url`.
- `cf-page-store answers <id>` prints the latest submission JSON. It prints nothing and exits 3 when no submission exists.

Both commands read the service-token id and secret from the OS keyring and never print either value.

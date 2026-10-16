#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
command_path="$PWD/cf-page-store/cf-page-store"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

failed=0
fail() {
  printf 'FAIL: %s: %s\n' "$1" "$2"
  failed=1
}

pass() {
  printf 'ok: %s\n' "$1"
}

export HOME="$tmp/home"
mkdir -p "$HOME/.agents/local" "$tmp/bin"
cat >"$HOME/.agents/local/cf-page-store.wrangler.jsonc" <<'JSON'
{"d1_databases": [{"database_name": "pages-db", "database_id": "db-id"}]}
JSON
cat >"$HOME/.agents/local/cf-page-store.md" <<EOF
Store address: https://store.example
Cloudflare account id: account-1
Team domain: https://team.example
Audience tag: audience
D1 database id: db-id
Deploy config: $HOME/.agents/local/cf-page-store.wrangler.jsonc
EOF
cat >"$tmp/bin/secret-tool" <<'SH'
#!/usr/bin/env bash
printf 'token-value'
SH
cat >"$tmp/bin/bunx" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >"$FAKE_BUNX_ARGS"
printf '%s\n' "token=$CLOUDFLARE_API_TOKEN account=$CLOUDFLARE_ACCOUNT_ID" >"$FAKE_BUNX_ENV"
if [[ -n "${FAKE_BUNX_FAIL:-}" ]]; then
  exit 1
fi
cat "$FAKE_BUNX_OUTPUT"
SH
chmod +x "$tmp/bin/secret-tool" "$tmp/bin/bunx"
export PATH="$tmp/bin:$PATH"
export FAKE_BUNX_ARGS="$tmp/bunx.args" FAKE_BUNX_ENV="$tmp/bunx.env" FAKE_BUNX_OUTPUT="$tmp/bunx.out"
cat >"$FAKE_BUNX_OUTPUT" <<'JSON'
[{"results": [{"id": "p1", "title": "One", "created_at": "2026-10-10 00:00:00", "herdr_tab_id": "w1:t1", "submitted_at": null}, {"id": "p2", "title": "Two", "created_at": "2026-10-11 00:00:00", "herdr_tab_id": null, "submitted_at": "2026-10-12 00:00:00"}], "success": true, "meta": {}}]
JSON

name='pages prints every row with its url'
status=0
output=$($command_path pages 2>&1) || status=$?
expected='[{"id":"p1","title":"One","created_at":"2026-10-10 00:00:00","herdr_tab_id":"w1:t1","submitted_at":null,"url":"https://store.example/p/p1"},{"id":"p2","title":"Two","created_at":"2026-10-11 00:00:00","herdr_tab_id":null,"submitted_at":"2026-10-12 00:00:00","url":"https://store.example/p/p2"}]'
if [[ "$status" != 0 ]]; then
  fail "$name" "exit $status: $output"
elif [[ "$(printf '%s' "$output" | jq -c .)" != "$expected" ]]; then
  fail "$name" "unexpected output: $output"
else
  pass "$name"
fi

name='pages runs a read-only wrangler query with the keyring token'
args=$(<"$FAKE_BUNX_ARGS")
env_line=$(<"$FAKE_BUNX_ENV")
if [[ "$args" != "--bun wrangler d1 execute pages-db --remote --config $HOME/.agents/local/cf-page-store.wrangler.jsonc --json --command SELECT pages.id"* ]]; then
  fail "$name" "unexpected wrangler arguments: $args"
elif [[ "$args" == *INSERT* || "$args" == *UPDATE* || "$args" == *DELETE* ]]; then
  fail "$name" "query is not read-only: $args"
elif [[ "$env_line" != 'token=token-value account=account-1' ]]; then
  fail "$name" "unexpected environment: $env_line"
else
  pass "$name"
fi

name='pages fails when the query fails'
status=0
output=$(FAKE_BUNX_FAIL=1 $command_path pages 2>&1) || status=$?
if [[ "$status" != 1 ]]; then
  fail "$name" "exit $status, want 1"
elif [[ "$output" != *'pages query failed'* ]]; then
  fail "$name" "missing reason in: $output"
else
  pass "$name"
fi

name='pages fails without a deploy config'
rm "$HOME/.agents/local/cf-page-store.wrangler.jsonc"
status=0
output=$($command_path pages 2>&1) || status=$?
if [[ "$status" != 1 ]]; then
  fail "$name" "exit $status, want 1"
elif [[ "$output" != *'existing deploy config'* ]]; then
  fail "$name" "missing reason in: $output"
else
  pass "$name"
fi

(( failed )) && exit 1
exit 0

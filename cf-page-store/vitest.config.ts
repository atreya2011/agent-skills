import { cloudflareTest, readD1Migrations } from "@cloudflare/vitest-pool-workers";
import { createServer } from "node:http";
import { exportJWK, exportPKCS8, generateKeyPair } from "jose";
import { defineConfig } from "vitest/config";

export default defineConfig(async () => {
  const { privateKey, publicKey } = await generateKeyPair("RS256", { extractable: true });
  const jwk = await exportJWK(publicKey);
  const keyServer = createServer((request, response) => {
    if (request.url !== "/cdn-cgi/access/certs") {
      response.writeHead(404).end();
      return;
    }
    response.setHeader("Content-Type", "application/json");
    response.end(JSON.stringify({ keys: [{ ...jwk, alg: "RS256", kid: "test-key", use: "sig" }] }));
  });
  await new Promise<void>((resolve) => keyServer.listen(0, "localhost", resolve));
  const address = keyServer.address();
  if (!address || typeof address === "string") {
    throw new Error("key server has no port");
  }
  const migrations = await readD1Migrations("./migrations");
  return {
    plugins: [
      {
        name: "access-key-server",
        configureVitest(context: import("vitest/node").VitestPluginContext) {
          context.vitest.onClose(() => new Promise<void>((resolve, reject) => {
            keyServer.close((error) => error ? reject(error) : resolve());
          }));
        },
      },
      cloudflareTest({
        wrangler: { configPath: "./wrangler.test.jsonc" },
        miniflare: {
          bindings: {
            TEST_MIGRATIONS: migrations,
            TEST_SIGNING_KEY: await exportPKCS8(privateKey),
            ACCESS_TEAM_DOMAIN: `http://localhost:${address.port}`,
          },
        },
      }),
    ],
  };
});

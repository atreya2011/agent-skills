import { cloudflareTest, readD1Migrations } from "@cloudflare/vitest-plugin";
import { exportJWK, exportPKCS8, generateKeyPair } from "jose";
import { defineConfig } from "vitest/config";

export default defineConfig(async () => {
  const { privateKey, publicKey } = await generateKeyPair("RS256", { extractable: true });
  const jwk = await exportJWK(publicKey);
  const keyServer = Bun.serve({
    hostname: "localhost",
    port: 0,
    fetch(request) {
      if (new URL(request.url).pathname !== "/cdn-cgi/access/certs") {
        return new Response(null, { status: 404 });
      }
      return Response.json({ keys: [{ ...jwk, alg: "RS256", kid: "test-key", use: "sig" }] });
    },
  });
  const migrations = await readD1Migrations("./migrations");
  return {
    test: {
      include: ["test/**/*.test.ts"],
      fsModuleCache: false,
    },
    plugins: [
      {
        name: "access-key-server",
        configureVitest(context: import("vitest/node").VitestPluginContext) {
          context.vitest.onClose(() => keyServer.stop(true));
        },
      },
      cloudflareTest({
        wrangler: { configPath: "./wrangler.test.jsonc" },
        miniflare: {
          bindings: {
            TEST_MIGRATIONS: migrations,
            TEST_SIGNING_KEY: await exportPKCS8(privateKey),
            ACCESS_TEAM_DOMAIN: `http://localhost:${keyServer.port}`,
          },
        },
      }),
    ],
  };
});

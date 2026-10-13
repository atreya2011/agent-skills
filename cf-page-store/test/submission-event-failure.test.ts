import { applyD1Migrations, createExecutionContext, runInDurableObject, waitOnExecutionContext } from "cloudflare:test";
import { env } from "cloudflare:workers";
import type { D1Migration } from "@cloudflare/vitest-plugin";
import { importPKCS8, SignJWT } from "jose";
import { beforeAll, expect, it } from "vitest";
import worker, { type PageEvents } from "../src/index";

declare global {
  namespace Cloudflare {
    interface Env {
      DB: D1Database;
      EVENTS: DurableObjectNamespace<PageEvents>;
      ASSETS: Fetcher;
      ACCESS_TEAM_DOMAIN: string;
      ACCESS_AUDIENCE: string;
      TEST_SIGNING_KEY: string;
      TEST_MIGRATIONS: D1Migration[];
    }
  }
}

beforeAll(async () => {
  await applyD1Migrations(env.DB, env.TEST_MIGRATIONS);
});

it.each(["wTEST:tR"])("keeps a saved answer successful when event storage fails for tab %s", async (tabId) => {
  const pageId = crypto.randomUUID();
  await env.DB.prepare("INSERT INTO pages (id, title, html, herdr_tab_id) VALUES (?, ?, ?, ?)")
    .bind(pageId, "Question", "<html><title>Question</title></html>", tabId).run();
  const now = Math.floor(Date.now() / 1000);
  const privateKey = await importPKCS8(env.TEST_SIGNING_KEY, "RS256");
  const assertion = await new SignJWT({})
    .setProtectedHeader({ alg: "RS256", kid: "test-key" })
    .setIssuer(env.ACCESS_TEAM_DOMAIN)
    .setAudience(env.ACCESS_AUDIENCE)
    .setIssuedAt(now)
    .setExpirationTime(now + 3600)
    .sign(privateKey);
  const events = env.EVENTS.getByName("listener");
  await runInDurableObject(events, (_instance, state) => {
    state.storage.sql.exec("CREATE TRIGGER reject_event BEFORE INSERT ON events BEGIN SELECT RAISE(FAIL, 'event storage unavailable'); END");
  });
  const context = createExecutionContext();
  try {
    const response = await worker.fetch(new Request(`https://store.invalid/p/${pageId}/answers`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "Cf-Access-Jwt-Assertion": assertion },
      body: '{"choice":"submitted"}',
    }), env, context);
    expect(response.status).toBe(204);
    await waitOnExecutionContext(context);
    expect(await env.DB.prepare("SELECT body FROM submissions WHERE page_id = ?")
      .bind(pageId).first()).toEqual({ body: '{"choice":"submitted"}' });
    expect(await runInDurableObject(events, (_instance, state) =>
      state.storage.sql.exec("SELECT id FROM events").toArray())).toEqual([]);
  } finally {
    await waitOnExecutionContext(context);
    await runInDurableObject(events, (_instance, state) => {
      state.storage.sql.exec("DROP TRIGGER reject_event");
    });
  }
});

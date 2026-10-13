import {
  applyD1Migrations,
  createExecutionContext,
  createScheduledController,
  evictDurableObject,
  runInDurableObject,
  waitOnExecutionContext,
} from "cloudflare:test";
import { env } from "cloudflare:workers";
import type { D1Migration } from "@cloudflare/vitest-pool-workers";
import { importPKCS8, SignJWT, type CryptoKey } from "jose";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";
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

    interface GlobalProps {
      mainModule: typeof import("../src/index");
    }
  }
}

const pageHeaders = {
  "content-security-policy": "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'",
  "cache-control": "no-store",
  "x-content-type-options": "nosniff",
  "referrer-policy": "no-referrer",
};
const html = "<!doctype html><html><head><title>Question</title></head><body>Choose</body></html>";

let privateKey: CryptoKey;

async function token(overrides: { audience?: string; issuer?: string; expired?: boolean } = {}) {
  const now = Math.floor(Date.now() / 1000);
  return new SignJWT({})
    .setProtectedHeader({ alg: "RS256", kid: "test-key" })
    .setIssuer(overrides.issuer ?? env.ACCESS_TEAM_DOMAIN)
    .setAudience(overrides.audience ?? env.ACCESS_AUDIENCE)
    .setIssuedAt(overrides.expired ? now - 120 : now)
    .setExpirationTime(overrides.expired ? now - 60 : now + 3600)
    .sign(privateKey);
}

async function request(path: string, init: RequestInit = {}, assertion?: string) {
  const headers = new Headers(init.headers);
  if (assertion) {
    headers.set("Cf-Access-Jwt-Assertion", assertion);
  }
  const context = createExecutionContext();
  const response = await worker.fetch(new Request(`https://store.invalid${path}`, { ...init, headers }), env, context);
  await waitOnExecutionContext(context);
  return response;
}

async function publish(assertion: string, tabId?: string) {
  const response = await request("/api/pages", {
    method: "POST",
    headers: { "Content-Type": "text/html", ...(tabId === undefined ? {} : { "X-Herdr-Tab-Id": tabId }) },
    body: html,
  }, assertion);
  expect(response.status).toBe(201);
  return response.json<{ id: string; url: string }>();
}

beforeAll(async () => {
  privateKey = await importPKCS8(env.TEST_SIGNING_KEY, "RS256");
  await applyD1Migrations(env.DB, env.TEST_MIGRATIONS);
});

describe("pages", () => {
  it("publishes and opens a page with the required headers", async () => {
    const assertion = await token();
    const page = await publish(assertion);
    expect(page.url).toBe(`https://store.invalid/p/${page.id}`);

    const response = await request(`/p/${page.id}`, {}, assertion);
    expect(response.status).toBe(200);
    expect(await response.text()).toBe(html);
    for (const [name, value] of Object.entries(pageHeaders)) {
      expect(response.headers.get(name)).toBe(value);
    }
  });

  it.each([undefined, "wTEST:t1", "wTEST2:t12", "wTEST:t0", "w2J:t1A", "w2H:tR"])("stores optional asking tab %s and submits without a listener", async (tabId) => {
    const assertion = await token();
    const page = await publish(assertion, tabId);
    expect(await env.DB.prepare("SELECT herdr_tab_id FROM pages WHERE id = ?")
      .bind(page.id).first()).toEqual({ herdr_tab_id: tabId ?? null });
    const response = await request(`/p/${page.id}/answers`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: '{"choice":"submitted"}',
    }, assertion);
    expect(response.status).toBe(204);
    expect(await env.DB.prepare("SELECT body FROM submissions WHERE page_id = ?")
      .bind(page.id).first()).toEqual({ body: '{"choice":"submitted"}' });
  });

  it.each(["", "wTEST", "wTEST:t-1", "wTEST:t1 extra", "wTEST:t1] @everyone", "../tab", "w_TEST:t1"])("rejects invalid asking tab %j without storing a page", async (tabId) => {
    const before = await env.DB.prepare("SELECT COUNT(*) AS count FROM pages").first();
    const response = await request("/api/pages", {
      method: "POST",
      headers: { "Content-Type": "text/html", "X-Herdr-Tab-Id": tabId },
      body: html,
    }, await token());
    expect(response.status).toBe(400);
    expect(await env.DB.prepare("SELECT COUNT(*) AS count FROM pages").first()).toEqual(before);
  });

  it.each([
    ["one-byte oversized page", "text/html", "x".repeat(1024 * 1024 + 1), 413],
    ["non-HTML page", "text/plain", html, 415],
  ])("rejects a %s", async (_name, contentType, body, status) => {
    const assertion = await token();
    const page = await publish(assertion);
    for (const [method, path] of [["POST", "/api/pages"], ["PUT", `/api/pages/${page.id}`]]) {
      const response = await request(path, { method, headers: { "Content-Type": contentType }, body }, assertion);
      expect(response.status, `${method} ${path}`).toBe(status);
    }
    expect(await (await request(`/p/${page.id}`, {}, assertion)).text()).toBe(html);
  });

  it.each([undefined, "wTEST:tU"])("replaces a page in place with asking tab %s and discards its submission", async (tabId) => {
    const assertion = await token();
    const page = await publish(assertion, tabId);
    expect((await request(`/p/${page.id}/answers`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: '{"choice":"old"}',
    }, assertion)).status).toBe(204);

    const replaced = "<!doctype html><html><head><title>Revised</title></head><body>Choose again</body></html>";
    const response = await request(`/api/pages/${page.id}`, {
      method: "PUT",
      headers: { "Content-Type": "text/html" },
      body: replaced,
    }, assertion);
    expect(response.status).toBe(200);
    expect(await response.json()).toEqual(page);

    expect(await env.DB.prepare("SELECT title, html, herdr_tab_id FROM pages WHERE id = ?").bind(page.id).first())
      .toEqual({ title: "Revised", html: replaced, herdr_tab_id: tabId ?? null });
    expect((await request(`/p/${page.id}`, {}, assertion)).status).toBe(200);
    expect((await request(`/p/${page.id}/answers`, {}, assertion)).status).toBe(404);
  });

  it.each(["unanswered", "answered"])("deletes an %s page", async (state) => {
    const assertion = await token();
    const page = await publish(assertion, "wTEST:tD");
    if (state === "answered") {
      expect((await request(`/p/${page.id}/answers`, {
        method: "POST", headers: { "Content-Type": "application/json" }, body: "{}",
      }, assertion)).status).toBe(204);
    }

    expect((await request(`/api/pages/${page.id}`, { method: "DELETE" }, assertion)).status).toBe(204);
    expect(await env.DB.prepare("SELECT id FROM pages WHERE id = ?").bind(page.id).first()).toBeNull();
    expect(await env.DB.prepare("SELECT page_id FROM submissions WHERE page_id = ?").bind(page.id).first()).toBeNull();
    expect((await request(`/p/${page.id}`, {}, assertion)).status).toBe(404);
    expect((await request(`/api/pages/${page.id}`, { method: "DELETE" }, assertion)).status).toBe(404);
  });
});

describe("assets", () => {
  it("serves a manifest asset through the assets binding", async () => {
    const response = await request("/assets/alpine-csp.min.js", {}, await token());
    expect(response.status).toBe(200);
    expect(await response.text()).toBe("test asset\n");
  });
});

describe("Access", () => {
  it.each([
    ["missing token", undefined],
    ["expired token", { expired: true }],
    ["wrong audience", { audience: "wrong-audience" }],
    ["wrong issuer", { issuer: "https://wrong.invalid" }],
  ])("returns 403 for a %s", async (_name, options) => {
    const assertion = options ? await token(options) : undefined;
    for (const [method, path] of [["POST", "/api/pages"], ["PUT", "/api/pages/any"], ["DELETE", "/api/pages/any"]]) {
      const response = await request(path, { method }, assertion);
      expect(response.status, `${method} ${path}`).toBe(403);
    }
  });
});

describe("submissions", () => {
  it("returns 404 before an answer and replaces the first answer", async () => {
    const assertion = await token();
    const page = await publish(assertion);
    expect((await request(`/p/${page.id}/answers`, {}, assertion)).status).toBe(404);

    for (const body of ['{"choice":"first"}', '{"choice":"final"}']) {
      const response = await request(`/p/${page.id}/answers`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body,
      }, assertion);
      expect(response.status).toBe(204);
    }

    const response = await request(`/p/${page.id}/answers`, {}, assertion);
    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ choice: "final" });
  });

  it.each([
    ["one-byte oversized submission", "application/json", `{"value":"${"x".repeat(65525)}"}`, 413],
    ["array submission", "application/json", "[]", 400],
    ["non-JSON submission", "text/plain", "{}", 400],
  ])("rejects a %s", async (_name, contentType, body, status) => {
    const assertion = await token();
    const page = await publish(assertion);
    const response = await request(`/p/${page.id}/answers`, {
      method: "POST",
      headers: { "Content-Type": contentType },
      body,
    }, assertion);
    expect(response.status).toBe(status);
  });
});

describe("scheduled cleanup", () => {
  it("deletes only pages whose last submission is more than three days old", async () => {
    const assertion = await token();
    const expired = await publish(assertion);
    const recent = await publish(assertion);
    const unanswered = await publish(assertion);

    for (const page of [expired, recent]) {
      const response = await request(`/p/${page.id}/answers`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: "{}",
      }, assertion);
      expect(response.status).toBe(204);
    }
    await env.DB.batch([
      env.DB.prepare("UPDATE submissions SET submitted_at = datetime('now', '-4 days') WHERE page_id = ?")
        .bind(expired.id),
      env.DB.prepare("UPDATE submissions SET submitted_at = datetime('now', '-1 day') WHERE page_id = ?")
        .bind(recent.id),
    ]);

    await worker.scheduled(createScheduledController(), env);
    expect(await env.DB.prepare("SELECT page_id FROM submissions WHERE page_id = ?")
      .bind(expired.id)
      .first()).toBeNull();

    for (const [page, pageStatus, answerStatus] of [
      [expired, 404, 404],
      [recent, 200, 200],
      [unanswered, 200, 404],
    ] as const) {
      expect((await request(`/p/${page.id}`, {}, assertion)).status).toBe(pageStatus);
      expect((await request(`/p/${page.id}/answers`, {}, assertion)).status).toBe(answerStatus);
    }
  });
});

describe("unknown pages", () => {
  it.each([
    ["GET", "/p/unknown", undefined, undefined],
    ["GET", "/p/unknown/answers", undefined, undefined],
    ["POST", "/p/unknown/answers", "application/json", "{}"],
    ["PUT", "/api/pages/unknown", "text/html", html],
    ["DELETE", "/api/pages/unknown", undefined, undefined],
  ])("returns 404 for %s %s", async (method, path, contentType, body) => {
    const response = await request(path, {
      method,
      headers: contentType ? { "Content-Type": contentType } : undefined,
      body,
    }, await token());
    expect(response.status).toBe(404);
  });
});


describe("answer events", () => {
  const sockets: WebSocket[] = [];
  const events = () => env.EVENTS.getByName("listener");

  beforeEach(async () => {
    await runInDurableObject(events(), (_instance, state) => {
      state.storage.sql.exec("DELETE FROM events");
    });
  });

  afterEach(() => {
    for (const socket of sockets.splice(0)) {
      socket.close(1000);
    }
  });

  async function connect(assertion: string) {
    const response = await request("/api/events", { headers: { Upgrade: "websocket" } }, assertion);
    expect(response.status).toBe(101);
    expect(response.webSocket).not.toBeNull();
    const socket = response.webSocket!;
    socket.accept();
    sockets.push(socket);
    return socket;
  }

  function receive(socket: WebSocket) {
    return new Promise<{ id: string; page_id: string; tab_id: string }>((resolve) => {
      socket.addEventListener("message", (message) => resolve(JSON.parse(message.data as string)), { once: true });
    });
  }

  async function storedEvents() {
    return runInDurableObject(events(), (_instance, state) =>
      state.storage.sql.exec("SELECT id, page_id, tab_id FROM events ORDER BY rowid").toArray());
  }

  it.each(["connected", "offline"])("delivers %s submissions, replays unacked events after hibernation, and deletes acknowledged events", async (mode) => {
    const assertion = await token();
    const page = await publish(assertion, "wTEST:tR");
    const first = mode === "connected" ? await connect(assertion) : undefined;
    const received = first ? receive(first) : undefined;
    const response = await request(`/p/${page.id}/answers`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: '{"choice":"submitted"}',
    }, assertion);
    expect(response.status).toBe(204);
    const saved = await storedEvents();
    expect(saved).toHaveLength(1);
    const event = saved[0];
    expect(event).toEqual({ id: expect.any(String), page_id: page.id, tab_id: "wTEST:tR" });
    expect(event.id).toMatch(/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/);
    if (received) {
      expect(await received).toEqual(event);
    }
    expect(await (await request(`/p/${page.id}/answers`, {}, assertion)).json()).toEqual({ choice: "submitted" });

    await evictDurableObject(events());
    const second = await connect(assertion);
    expect(await receive(second)).toEqual(event);
    second.send(JSON.stringify({ ack: event.id }));
    await expect.poll(storedEvents).toEqual([]);
    const third = await connect(assertion);
    expect(third.readyState).toBe(WebSocket.OPEN);
    expect(await storedEvents()).toEqual([]);
  });

  it.each([1, 2])("broadcasts each submission to %s connected sockets", async (count) => {
    const assertion = await token();
    const page = await publish(assertion, "wTEST:tA");
    const listeners = await Promise.all(Array.from({ length: count }, () => connect(assertion)));
    const received = listeners.map(receive);
    for (const choice of ["first", "second"]) {
      const messages = choice === "first" ? received : listeners.map(receive);
      expect((await request(`/p/${page.id}/answers`, {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ choice }),
      }, assertion)).status).toBe(204);
      const delivered = await Promise.all(messages);
      expect(new Set(delivered.map((event) => event.id)).size).toBe(1);
      for (const event of delivered) {
        expect(event).toEqual({ id: expect.any(String), page_id: page.id, tab_id: "wTEST:tA" });
      }
    }
    expect(await storedEvents()).toHaveLength(2);
  });

  it.each([undefined])("does not queue an event without an asking tab", async (tabId) => {
    const assertion = await token();
    const page = await publish(assertion, tabId);
    expect((await request(`/p/${page.id}/answers`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: "{}",
    }, assertion)).status).toBe(204);
    expect(await storedEvents()).toEqual([]);
  });

  it.each([
    ["missing token", undefined, "websocket", 403],
    ["expired token", { expired: true }, "websocket", 403],
    ["wrong audience", { audience: "wrong-audience" }, "websocket", 403],
    ["no upgrade", {}, undefined, 426],
  ] as const)("rejects an event connection with %s", async (_name, options, upgrade, status) => {
    const response = await request("/api/events", {
      headers: upgrade ? { Upgrade: upgrade } : {},
    }, options ? await token(options) : undefined);
    expect(response.status).toBe(status);
  });
});

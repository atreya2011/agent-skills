import { applyD1Migrations } from "cloudflare:test";
import { env, exports } from "cloudflare:workers";
import type { D1Migration } from "@cloudflare/vitest-pool-workers";
import { setupNetwork } from "@msw/cloudflare";
import { HttpResponse, http } from "msw";
import { exportJWK, generateKeyPair, SignJWT, type CryptoKey } from "jose";
import { afterAll, beforeAll, describe, expect, it } from "vitest";

declare global {
  namespace Cloudflare {
    interface Env {
      DB: D1Database;
      ACCESS_TEAM_DOMAIN: string;
      ACCESS_AUDIENCE: string;
      TEST_MIGRATIONS: D1Migration[];
    }

    interface GlobalProps {
      mainModule: typeof import("../src/index");
    }
  }
}

const network = setupNetwork();
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
  return exports.default.fetch(`https://store.invalid${path}`, { ...init, headers });
}

async function publish(assertion: string) {
  const response = await request("/api/pages", {
    method: "POST",
    headers: { "Content-Type": "text/html" },
    body: html,
  }, assertion);
  expect(response.status).toBe(201);
  return response.json<{ id: string; url: string }>();
}

beforeAll(async () => {
  const pair = await generateKeyPair("RS256");
  privateKey = pair.privateKey;
  const publicKey = await exportJWK(pair.publicKey);
  network.use(http.get(`${env.ACCESS_TEAM_DOMAIN}/cdn-cgi/access/certs`, () => HttpResponse.json({
    keys: [{ ...publicKey, alg: "RS256", kid: "test-key", use: "sig" }],
  })));
  network.enable();
  await applyD1Migrations(env.DB, env.TEST_MIGRATIONS);
});

afterAll(() => network.disable());

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

  it.each([
    ["one-byte oversized page", "text/html", "x".repeat(1024 * 1024 + 1), 413],
    ["non-HTML page", "text/plain", html, 415],
  ])("rejects a %s", async (_name, contentType, body, status) => {
    const response = await request("/api/pages", {
      method: "POST",
      headers: { "Content-Type": contentType },
      body,
    }, await token());
    expect(response.status).toBe(status);
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
    const response = await request("/api/pages", { method: "POST" }, assertion);
    expect(response.status).toBe(403);
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
    expect(new TextEncoder().encode(body).byteLength).toBe(status === 413 ? 65537 : 2);
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

describe("unknown pages", () => {
  it.each([
    ["GET", "/p/unknown"],
    ["GET", "/p/unknown/answers"],
    ["POST", "/p/unknown/answers"],
  ])("returns 404 for %s %s", async (method, path) => {
    const response = await request(path, {
      method,
      headers: method === "POST" ? { "Content-Type": "application/json" } : undefined,
      body: method === "POST" ? "{}" : undefined,
    }, await token());
    expect(response.status).toBe(404);
  });
});

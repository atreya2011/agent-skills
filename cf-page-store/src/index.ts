import { Hono } from "hono";
import { createRemoteJWKSet, jwtVerify } from "jose";
import manifest from "../../phone-page/assets.json";

interface Env {
  DB: D1Database;
  ASSETS: Fetcher;
  ACCESS_TEAM_DOMAIN: string;
  ACCESS_AUDIENCE: string;
}

const pageLimit = 1024 * 1024;
const submissionLimit = 64 * 1024;
const assetNames = new Set(manifest.assets.map((asset) => asset.name));
const pageHeaders = {
  "Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'",
  "Cache-Control": "no-store",
  "X-Content-Type-Options": "nosniff",
  "Referrer-Policy": "no-referrer",
};

let keySet: ReturnType<typeof createRemoteJWKSet> | undefined;

const app = new Hono<{ Bindings: Env }>();

app.use("*", async (c, next) => {
  const token = c.req.header("Cf-Access-Jwt-Assertion");
  if (!token) {
    return c.body(null, 403);
  }

  try {
    keySet ??= createRemoteJWKSet(new URL(`${c.env.ACCESS_TEAM_DOMAIN}/cdn-cgi/access/certs`));
    await jwtVerify(token, keySet, {
      issuer: c.env.ACCESS_TEAM_DOMAIN,
      audience: c.env.ACCESS_AUDIENCE,
      algorithms: ["RS256"],
    });
  } catch {
    return c.body(null, 403);
  }

  await next();
});

app.post("/api/pages", async (c) => {
  const body = await c.req.arrayBuffer();
  if (body.byteLength > pageLimit) {
    return c.body(null, 413);
  }
  if (c.req.header("Content-Type")?.split(";", 1)[0].trim().toLowerCase() !== "text/html") {
    return c.body(null, 415);
  }

  const html = new TextDecoder().decode(body);
  const title = html.match(/<title(?:\s[^>]*)?>([\s\S]*?)<\/title>/i)?.[1].trim() ?? "";
  const id = crypto.randomUUID();
  await c.env.DB.prepare("INSERT INTO pages (id, title, html) VALUES (?, ?, ?)")
    .bind(id, title, html)
    .run();

  return c.json({ id, url: new URL(`/p/${id}`, c.req.url).href }, 201);
});

app.get("/p/:id", async (c) => {
  const page = await c.env.DB.prepare("SELECT html FROM pages WHERE id = ?")
    .bind(c.req.param("id"))
    .first<{ html: string }>();
  if (!page) {
    return c.body(null, 404);
  }
  return c.body(page.html, 200, { ...pageHeaders, "Content-Type": "text/html; charset=UTF-8" });
});

app.get("/p/:id/answers", async (c) => {
  const page = await c.env.DB.prepare("SELECT id FROM pages WHERE id = ?")
    .bind(c.req.param("id"))
    .first();
  if (!page) {
    return c.body(null, 404);
  }

  const submission = await c.env.DB.prepare("SELECT body FROM submissions WHERE page_id = ?")
    .bind(c.req.param("id"))
    .first<{ body: string }>();
  if (!submission) {
    return c.body(null, 404);
  }
  return c.body(submission.body, 200, { "Content-Type": "application/json" });
});

app.post("/p/:id/answers", async (c) => {
  const page = await c.env.DB.prepare("SELECT id FROM pages WHERE id = ?")
    .bind(c.req.param("id"))
    .first();
  if (!page) {
    return c.body(null, 404);
  }

  const body = await c.req.arrayBuffer();
  if (body.byteLength > submissionLimit) {
    return c.body(null, 413);
  }
  if (c.req.header("Content-Type")?.split(";", 1)[0].trim().toLowerCase() !== "application/json") {
    return c.body(null, 400);
  }

  const text = new TextDecoder().decode(body);
  try {
    const submission: unknown = JSON.parse(text);
    if (typeof submission !== "object" || submission === null || Array.isArray(submission)) {
      return c.body(null, 400);
    }
  } catch {
    return c.body(null, 400);
  }

  await c.env.DB.prepare(`
    INSERT INTO submissions (page_id, body) VALUES (?, ?)
    ON CONFLICT(page_id) DO UPDATE SET body = excluded.body, submitted_at = CURRENT_TIMESTAMP
  `)
    .bind(c.req.param("id"), text)
    .run();
  return c.body(null, 204);
});

app.get("/assets/:name", async (c) => {
  if (!assetNames.has(c.req.param("name"))) {
    return c.body(null, 404);
  }
  return c.env.ASSETS.fetch(new URL(`/${c.req.param("name")}`, c.req.url));
});

export default app;

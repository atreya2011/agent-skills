import { describe, expect, it } from "bun:test";

describe("listener heartbeat", () => {
  it.each([
    { name: "keeps a responsive connection and ignores text pongs", reply: true },
    { name: "terminates a silent connection and reconnects after the pong deadline", reply: false },
  ])("$name", async ({ reply }) => {
    const connections = [];
    const messages = [];
    const closed = [];
    const completed = Promise.withResolvers();
    const server = Bun.serve({
      hostname: "localhost",
      port: 0,
      fetch(request, server) {
        if (server.upgrade(request)) {
          return;
        }
        return new Response(null, { status: 400 });
      },
      websocket: {
        open(socket) {
          connections.push({ socket, at: performance.now() });
          if (connections.length === 2) {
            completed.resolve("reconnected");
          }
        },
        message(socket, data) {
          messages.push({ data, at: performance.now() });
          if (reply && data === "ping") {
            socket.send("pong");
            if (messages.length === 2) {
              completed.resolve("responsive");
            }
          }
        },
        close(socket, code) {
          closed.push({ socket, code, at: performance.now() });
        },
      },
    });
    const listener = Bun.spawn([process.execPath, `${import.meta.dir}/../listen.mjs`, server.url.origin], {
      stdin: new Blob(["test-client\ntest-secret\n"]),
      stdout: "ignore",
      stderr: "pipe",
    });
    let errors;
    try {
      expect(await completed.promise).toBe(reply ? "responsive" : "reconnected");
      expect(messages.map(({ data }) => data)).toEqual(reply ? ["ping", "ping"] : ["ping"]);
      expect(messages[0].at - connections[0].at).toBeGreaterThanOrEqual(29500);
      expect(messages[0].at - connections[0].at).toBeLessThan(35000);
      if (reply) {
        expect(connections).toHaveLength(1);
        expect(closed).toEqual([]);
        expect(messages[1].at - messages[0].at).toBeGreaterThanOrEqual(29500);
        expect(messages[1].at - messages[0].at).toBeLessThan(35000);
      } else {
        expect(connections).toHaveLength(2);
        expect(closed).toHaveLength(1);
        expect(closed[0].socket).toBe(connections[0].socket);
        expect(closed[0].code).toBe(1006);
        expect(closed[0].at - messages[0].at).toBeGreaterThanOrEqual(9500);
        expect(closed[0].at - messages[0].at).toBeLessThan(15000);
        expect(connections[1].at - closed[0].at).toBeGreaterThanOrEqual(950);
      }
    } finally {
      listener.kill();
      await listener.exited;
      errors = await new Response(listener.stderr).text();
      await server.stop(true);
    }
    expect(errors).toBe(reply ? "" : "page store listener: disconnected; reconnecting\n");
  }, 75000);
});

import { execFile } from "node:child_process";
import { readFileSync } from "node:fs";
import { promisify } from "node:util";
import WebSocket from "ws";

const run = promisify(execFile);
const tabPattern = /^w[A-Za-z0-9]+:t[A-Za-z0-9]+$/;
const uuidPattern = /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/i;
const [clientId, clientSecret] = readFileSync(0, "utf8").trimEnd().split("\n");
if (!clientId || !clientSecret) {
  console.error("page store listener: missing service-token credentials");
  process.exit(1);
}
const url = new URL("/api/events", process.argv[2]);
url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
let backoff = 1000;

async function handleEvent(socket, data) {
  let event;
  try {
    event = JSON.parse(data.toString());
  } catch {
    console.error("page store listener: invalid event");
    return;
  }
  if (!event || typeof event.id !== "string" || !uuidPattern.test(event.id)) {
    console.error(`page store listener: invalid event id for page ${JSON.stringify(event?.page_id)} tab ${JSON.stringify(event?.tab_id)}`);
    return;
  }
  if (typeof event.page_id !== "string" || !uuidPattern.test(event.page_id)
    || typeof event.tab_id !== "string" || !tabPattern.test(event.tab_id)) {
    console.error(`page store listener: invalid page or tab id for page ${JSON.stringify(event.page_id)} tab ${JSON.stringify(event.tab_id)}`);
  } else {
    try {
      const { stdout } = await run("herdr", ["agent", "list"]);
      const agents = JSON.parse(stdout).result.agents.filter((agent) => agent.tab_id === event.tab_id);
      if (agents.length === 1 && typeof agents[0].pane_id === "string"
        && ["idle", "done", "working"].includes(agents[0].agent_status)) {
        await run("herdr", ["agent", "prompt", agents[0].pane_id,
          `AGENT MESSAGE (page store, not a human). Page ${event.page_id} has a new submission. Read it with phone-page answers ${event.page_id}.`]);
      } else {
        console.error(`page store listener: no eligible agent for page ${JSON.stringify(event.page_id)} tab ${JSON.stringify(event.tab_id)}`);
      }
    } catch {
      console.error(`page store listener: could not prompt page ${JSON.stringify(event.page_id)} tab ${JSON.stringify(event.tab_id)}`);
    }
  }
  if (socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ ack: event.id }));
  }
}

function connect() {
  const socket = new WebSocket(url, {
    headers: {
      "CF-Access-Client-Id": clientId,
      "CF-Access-Client-Secret": clientSecret,
    },
    followRedirects: false,
  });
  let reconnecting = false;
  let pending = Promise.resolve();
  const reconnect = () => {
    if (reconnecting) {
      return;
    }
    reconnecting = true;
    socket.terminate();
    console.error("page store listener: disconnected; reconnecting");
    setTimeout(connect, backoff);
    backoff = Math.min(backoff * 2, 60000);
  };
  socket.on("open", () => { backoff = 1000; });
  socket.on("message", (data) => {
    pending = pending.then(() => handleEvent(socket, data)).catch(() => {
      console.error("page store listener: event processing failed");
    });
  });
  socket.on("close", reconnect);
  socket.on("error", reconnect);
}

connect();

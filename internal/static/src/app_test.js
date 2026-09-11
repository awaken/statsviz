import assert from "node:assert/strict";
import test from "node:test";

import { loadModule } from "./vm_test_helper.js";

async function loadApp() {
  const frames = [];
  const updates = [];
  let navUpdate;
  let socket;

  class StatsManager {
    pushData() {}
    slice() { return {}; }
  }
  class PlotManager {
    constructor() { this.plots = []; }
    update(_data, _gcEnabled, _timerange, force) { updates.push(force); }
  }
  class WebSocketClient {
    constructor(_uri, onConfig, onData) {
      socket = { onConfig, onData };
    }
  }

  const { namespace } = await loadModule(new URL("./app.js", import.meta.url), {
    "./PlotManager.js": { default: PlotManager },
    "./StatsManager.js": { default: StatsManager },
    "./nav.js": {
      gcEnabled: true,
      initNav: (fn) => { navUpdate = fn; },
      running: true,
      timerange: 60,
    },
    "./socket.js": { default: WebSocketClient },
    "./utils.js": { buildWebsocketURI: () => "ws://example/ws" },
  }, {
    requestAnimationFrame: (fn) => {
      frames.push(fn);
      return frames.length;
    },
  });

  namespace.connect();
  socket.onConfig({ events: [], series: [] });

  return { frames, navUpdate: () => navUpdate(true), socket, updates };
}

test("ordinary samples preserve each plot update frequency", async () => {
  const app = await loadApp();

  app.socket.onData({ series: {}, timestamp: 1 });
  app.frames.shift()();

  assert.deepEqual(app.updates, [false]);
});

test("navigation changes force an immediate plot redraw", async () => {
  const app = await loadApp();

  app.navUpdate();
  app.frames.shift()();

  assert.deepEqual(app.updates, [true]);
});

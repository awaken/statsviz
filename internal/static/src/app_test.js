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
      updateVisibility() {},
      running: true,
      timerange: 60,
    },
    "./socket.js": { default: WebSocketClient },
    "./utils.js": { buildWebsocketURI: () => "ws://example/ws" },
  }, {
    document: {getElementById: () => null},
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

test("audit: manual redraw still works while live updates are paused", async () => {
  const frames = [];
  const slices = [];
  const updates = [];
  let socket;
  const { namespace } = await loadModule(new URL("./app.js", import.meta.url), {
    "./PlotManager.js": {default: class {constructor() {this.plots=[];} update(...args) {updates.push(args);} dispose() {}}},
    "./StatsManager.js": {default: class {pushData() {} slice(range) {slices.push(range); return {times:[1]};}}},
    "./nav.js": {gcEnabled:true, initNav() {}, updateVisibility() {}, running:false, timerange:60},
    "./socket.js": {default: class {constructor(_uri, onConfig, onData) {socket={onConfig,onData};}}},
    "./utils.js": {buildWebsocketURI: () => "ws://owned.test/ws"},
  }, {
    document: {getElementById: () => ({hidden:false})},
    requestAnimationFrame: fn => (frames.push(fn), frames.length),
    cancelAnimationFrame() {},
  });
  namespace.connect();
  socket.onConfig({series:[],events:[]});
  socket.onData({});
  frames.shift()();
  assert.equal(updates.length, 0, "ordinary live sample advanced a paused plot");
  namespace.drawPlots(true);
  frames.shift()();
  assert.equal(updates.length, 1, "time-range/GC controls did not redraw the paused plot");
  namespace.drawPlots(true);
  frames.shift()();
  assert.deepEqual(slices, [Infinity], "paused controls did not reuse one frozen history snapshot");
});

test("reconfiguration disposes old plots and cancels queued drawing", async () => {
  const frames = new Map();
  let id = 0, socket, disposed = 0, nav = 0, visibility = 0;
  const {namespace} = await loadModule(new URL("./app.js", import.meta.url), {
    "./PlotManager.js": {default: class {constructor() {this.plots=[];} update() {} dispose() {disposed++;}}},
    "./StatsManager.js": {default: class {pushData() {} slice() {return {historyLimited:true};}}},
    "./nav.js": {gcEnabled:true, initNav() {nav++;}, updateVisibility() {visibility++;}, running:true, timerange:60},
    "./socket.js": {default: class {constructor(_uri, onConfig, onData) {socket={onConfig,onData};}}},
    "./utils.js": {buildWebsocketURI: () => "ws://owned.test/ws"},
  }, {
    document: {getElementById: () => ({hidden:false})},
    requestAnimationFrame: fn => (frames.set(++id, fn), id),
    cancelAnimationFrame: i => frames.delete(i),
  });
  namespace.connect(); socket.onConfig({});
  namespace.drawPlots(true); namespace.drawPlots(false);
  assert.equal(frames.size, 1);
  socket.onConfig({});
  assert.equal(frames.size, 0); assert.equal(disposed, 1); assert.equal(nav, 1); assert.equal(visibility, 1);
  socket.onData({}); const callback = frames.values().next().value; frames.clear(); callback();
});

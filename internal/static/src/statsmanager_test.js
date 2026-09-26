import assert from "node:assert/strict";
import test from "node:test";
import StatsManager from "./StatsManager.js";

const config = {series: [{name: "user", type: "scatter", subplots: [{}]}], events: ["lastgc"]};
const push = (m, timestamp, value, event = timestamp) => m.pushData({timestamp, series: {user: [value], lastgc: [event]}});

test("audit: null samples remain gaps through history wrap and recovery", () => {
  const m = new StatsManager(600, config, {maxSamples: 3});
  push(m, 1000, 1);
  push(m, 2000, null);
  push(m, 3000, 42);
  push(m, 4000, 0);
  const values = Array.from(m.slice(60).series.get("user")[0]);
  assert.ok(values[0] === null || Number.isNaN(values[0]), `gap became ${values[0]}`);
  assert.deepEqual(values.slice(1), [42, 0]);
  m.pushData({timestamp: 5000, series: {lastgc: [5000]}});
  assert.ok(Number.isNaN(m.slice(60).series.get("user")[0].at(-1)), "missing sample became data");
});

test("history limits, retention, events, and clock corrections stay aligned", () => {
  assert.throws(() => new StatsManager(0, config));
  assert.throws(() => new StatsManager(Infinity, config));
  assert.throws(() => new StatsManager(10, config, {maxSamples: 0}));
  assert.throws(() => new StatsManager(10, config, {maxValues: 1}));
  const m = new StatsManager(2, config, {maxSamples: 3, maxValues: 100});
  assert.equal(m.slice(10).times.length, 0);
  push(m, 1000, 1); push(m, 1500, 2, 1000); push(m, 2000, 3); push(m, 2500, 4);
  assert.deepEqual(Array.from(m.slice(10).times), [1500, 2000, 2500]);
  assert.equal(m.slice(10).historyLimited, true);
  assert.deepEqual(Array.from(m.slice(10).events.get("lastgc"), Number), [2000, 2500]);
  push(m, 5000, 5, -1000);
  assert.deepEqual(Array.from(m.slice(10).times), [5000]);
  assert.equal(m.slice(10).historyLimited, false);
  push(m, 100, 6, 200);
  assert.deepEqual(Array.from(m.slice(10).series.get("user")[0]), [6]);
  assert.equal(m.slice(10).events.get("lastgc").length, 0);
  assert.throws(() => push(m, NaN, 7));
  const heatmap = new StatsManager(1, {series: [{name: "user", type: "heatmap", buckets: [0]}], events: ["lastgc"]});
  push(heatmap, 100, 8);
  assert.deepEqual(Array.from(heatmap.slice(1).series.get("user")[0]), [8]);
});

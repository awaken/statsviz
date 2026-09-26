import assert from "node:assert/strict";
import test from "node:test";
import RingBuffer from "./RingBuffer.js";

test("ring buffer preserves order across wrap, drop, and bounded copies", () => {
  for (const n of [0, -1, 1.2, NaN]) assert.throws(() => new RingBuffer(n));
  const b = new RingBuffer(3);
  assert.equal(b.first, undefined); assert.equal(b.last, undefined);
  assert.equal(b.at(-1), undefined);
  assert.deepEqual(Array.from(b.slice(9)), []);
  b.push(1); b.push(2);
  assert.deepEqual(Array.from(b.slice(9)), [1, 2]);
  b.push(3); b.push(4);
  assert.deepEqual(Array.from(b.slice(3)), [2, 3, 4]);
  assert.equal(b.lowerBound(3), 1); assert.equal(b.lowerBound(10), 3);
  b.drop(1); b.push(5);
  assert.deepEqual(Array.from(b.slice(3)), [3, 4, 5]);
  b.drop(99); assert.equal(b.length, 0); b.push(7); b.drop(-1);
  assert.equal(b.first, 7);
});

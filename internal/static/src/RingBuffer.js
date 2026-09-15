export default class RingBuffer {
  #buf;
  #size = 0;
  #start = 0;

  constructor(capacity) {
    if (!Number.isSafeInteger(capacity) || capacity < 1) throw new Error("Capacity must be > 0");
    this.#buf = new Float64Array(capacity);
  }

  push(item) {
    const end = (this.#start + this.#size) % this.#buf.length;
    this.#buf[end] = item;
    if (this.#size < this.#buf.length) {
      this.#size++;
    } else {
      this.#start = (this.#start + 1) % this.#buf.length;
    }
  }

  slice(lastN) {
    const n = Math.min(lastN, this.#size);
    const result = new Float64Array(n);

    const source = (this.#start + this.#size - n) % this.#buf.length;
    const beforeWrap = Math.min(n, this.#buf.length - source);
    result.set(this.#buf.subarray(source, source + beforeWrap));
    if (beforeWrap < n) {
      result.set(this.#buf.subarray(0, n - beforeWrap), beforeWrap);
    }

    return result;
  }

  get length() { return this.#size; }

  at(index) {
    if (index < 0 || index >= this.#size) return undefined;
    return this.#buf[(this.#start + index) % this.#buf.length];
  }

  drop(count) {
    const n = Math.min(Math.max(0, count), this.#size);
    this.#start = (this.#start + n) % this.#buf.length;
    this.#size -= n;
  }

  // Timestamps are ordered; find the retained suffix without copying it.
  lowerBound(timestamp) {
    let lo = 0, hi = this.#size;
    while (lo < hi) {
      const mid = lo + Math.floor((hi - lo) / 2);
      if (this.at(mid) < timestamp) lo = mid + 1;
      else hi = mid;
    }
    return lo;
  }

  get last() { return this.at(this.#size - 1); }

  get first() {
    if (this.#size === 0) return undefined;
    return this.#buf[this.#start];
  }
}

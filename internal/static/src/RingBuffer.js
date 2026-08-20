export default class RingBuffer {
  #buf;
  #size = 0;
  #start = 0;

  constructor(capacity) {
    if (capacity < 1) throw new Error("Capacity must be > 0");
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

  get first() {
    if (this.#size === 0) return undefined;
    return this.#buf[this.#start];
  }
}

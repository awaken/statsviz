import RingBuffer from "./RingBuffer.js";

// Numeric history uses at most 16 MB and 60,000 samples, whichever is smaller.
const defaultLimits = {maxValues: 2000000, maxSamples: 60000};

export default class StatsManager {
  #retention;
  #limits;
  #capacity;
  #config;
  #times;
  #plotData;
  #eventsData;
  #droppedAt;

  constructor(retentionSeconds, config, limits = {}) {
    if (!Number.isFinite(retentionSeconds) || retentionSeconds <= 0) {
      throw new Error("Retention must be positive seconds");
    }
    this.#retention = retentionSeconds * 1000;
    this.#limits = {...defaultLimits, ...limits};
    for (const value of Object.values(this.#limits)) {
      if (!Number.isSafeInteger(value) || value < 1) throw new Error("Invalid history limit");
    }
    this.reset(config);
  }

  reset(config) {
    this.#config = config;
    const dims = config.series.map(pd => pd.type === "heatmap" ? pd.buckets.length : pd.subplots.length);
    const width = 1 + config.events.length + dims.reduce((a, b) => a + b, 0);
    if (width > this.#limits.maxValues) throw new Error("History limit cannot hold one sample");
    this.#capacity = Math.min(this.#limits.maxSamples, Math.floor(this.#limits.maxValues / width));
    this.#times = new RingBuffer(this.#capacity);
    this.#plotData = new Map(config.series.map((pd, i) => [pd.name,
      Array.from({length: dims[i]}, () => new RingBuffer(this.#capacity))]));
    this.#eventsData = new Map(config.events.map(evt => [evt, new RingBuffer(this.#capacity)]));
    this.#droppedAt = -Infinity;
  }

  pushData(payload) {
    const now = payload.timestamp;
    if (!Number.isFinite(now)) throw new Error("Invalid sample timestamp");
    // A server clock correction starts a new ordered history epoch.
    if (this.#times.length && now < this.#times.last) this.reset(this.#config);
    const expired = this.#times.lowerBound(now - this.#retention);
    this.#times.drop(expired);
    for (const buffers of this.#plotData.values()) {
      for (const buffer of buffers) buffer.drop(expired);
    }
    if (this.#times.length === this.#capacity) this.#droppedAt = this.#times.first;
    this.#times.push(now);
    for (const [name, buffers] of this.#plotData) {
      buffers.forEach((buffer, i) => buffer.push(payload.series[name][i]));
    }
    const oldest = this.#times.first;
    for (const [name, events] of this.#eventsData) {
      events.drop(events.lowerBound(oldest));
      const timestamp = Math.floor(payload.series[name][0]);
      if (Number.isFinite(timestamp) && timestamp >= oldest && timestamp <= now &&
          (!events.length || timestamp > events.last)) events.push(timestamp);
    }
  }

  slice(seconds) {
    const cutoff = this.#times.last - Math.min(seconds * 1000, this.#retention);
    const count = this.#times.length - this.#times.lowerBound(cutoff);
    const times = this.#times.slice(count);
    const series = new Map();
    for (const [name, buffers] of this.#plotData) {
      series.set(name, buffers.map(buffer => buffer.slice(count)));
    }
    const events = new Map();
    for (const [name, buffer] of this.#eventsData) {
      events.set(name, Array.from(buffer.slice(buffer.length - buffer.lowerBound(cutoff)), t => new Date(t)));
    }
    return {times, series, events, historyLimited: this.#droppedAt >= cutoff};
  }
}

import assert from "node:assert/strict";
import test from "node:test";

import { loadModule } from "./vm_test_helper.js";

async function loadPlot(plotly = {}, globals = {}) {
  return loadModule(new URL("./plot.js", import.meta.url), {
    "./plotConfig.js": {
      defaultPlotHeight: 480,
      newConfigObject: () => ({}),
      newLayoutObject: () => ({ xaxis: {} }),
      themeColors: {
        light: { font_color: "black", paper_bgcolor: "white", plot_bgcolor: "white" },
      },
    },
    "./theme.js": { getThemeMode: () => "light" },
    "./utils.js": { formatFunction: () => String },
    "plotly.js-cartesian-dist": { default: plotly },
  }, {
    document: { getElementById: () => ({}) },
    ...globals,
  });
}

test("plot search includes its displayed and stable names", async () => {
  const { namespace } = await loadPlot();
  const plot = new namespace.Plot({
    infoText: "",
    layout: {},
    metrics: ["/runtime/value:things"],
    name: "collector-activity",
    subplots: [],
    tags: ["gc"],
    title: "Collector Activity",
    type: "scatter",
  });

  assert.equal(plot.matches("COLLECTOR ACTIVITY"), true);
  assert.equal(plot.matches("collector-activity"), true);
  assert.equal(plot.matches("runtime/value"), true);
  assert.equal(plot.matches("unrelated"), false);
});

for (const [minimum, firstLabel] of [[undefined, "(-Inf, 8)"], [0, "[0, 8)"]]) {
  test(`heatmap hover preserves bucket bounds with minimum ${minimum}`, async () => {
    let trace;
    const { namespace } = await loadPlot({
      newPlot() {},
      react(_element, data) { trace = data[0]; },
    }, {
      IntersectionObserver: class {
        constructor(callback) { this.callback = callback; }
        observe() { this.callback([{ isIntersecting: true }]); }
      },
    });
    const plot = new namespace.Plot({
      buckets: [0, 1, 2],
      colorscale: [],
      custom_data: [8, 16, null],
      events: "",
      hover: { yname: "size class", ymin: minimum, yunit: "bytes", zname: "objects" },
      infoText: "",
      layout: {},
      name: "size-classes",
      type: "heatmap",
    });
    plot.createElement({ clientWidth: 640 });

    for (const times of [[0, 1], [0, 1, 2]]) {
      const counts = [[1, 2, 3], [4, 5, 6], [7, 8, 9]].map((row) => row.slice(0, times.length));
      plot.update([0, 2], {
        times,
        series: new Map([["size-classes", counts]]),
      }, new Map(), true);

      assert.deepEqual(Array.from(trace.text, (row) => Array.from(row)), [
        Array(times.length).fill(firstLabel),
        Array(times.length).fill("[8, 16)"),
        Array(times.length).fill("[16, +Inf)"),
      ]);
      assert.equal(trace.z, counts);
      assert.equal(trace.hoverongaps, false);
      assert.match(trace.hovertemplate, /size class.*%\{text\}.*objects.*%\{z\}/);
    }
  });
}

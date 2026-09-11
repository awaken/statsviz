import assert from "node:assert/strict";
import test from "node:test";

import { loadModule } from "./vm_test_helper.js";

test("plot search includes its displayed and stable names", async () => {
  const { namespace } = await loadModule(new URL("./plot.js", import.meta.url), {
    "./plotConfig.js": {
      defaultPlotHeight: 480,
      newConfigObject: () => ({}),
      newLayoutObject: () => ({}),
      themeColors: {
        light: { font_color: "black", paper_bgcolor: "white", plot_bgcolor: "white" },
      },
    },
    "./theme.js": { getThemeMode: () => "light" },
    "./utils.js": { formatFunction: () => String },
    "bootstrap-icons/font/bootstrap-icons.min.css": {},
    "plotly.js-cartesian-dist": { default: {} },
    "tippy.js": { default: () => ({}), followCursor: {} },
    "tippy.js/dist/tippy.css": {},
  }, {
    document: { getElementById: () => ({}) },
  });
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

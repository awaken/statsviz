import StatsManager from "./StatsManager.js";
import PlotManager from "./PlotManager.js";
import { initNav, updateVisibility, running, gcEnabled, timerange } from "./nav.js";
import { buildWebsocketURI } from "./utils.js";
import WebSocketClient from "./socket.js";

export let statsMgr;
export let plotMgr;

// RAF-based throttling for plot updates
let rafId = null;
let navReady = false;
let pendingUpdate = false;
let forceNextUpdate = false;

const scheduleUpdate = () => {
  if (rafId !== null) return; // Already scheduled

  rafId = requestAnimationFrame(() => {
    rafId = null;
    if (pendingUpdate && running) {
      const data = statsMgr.slice(timerange);
      const warning = document.getElementById("history-warning");
      if (warning) warning.hidden = !data.historyLimited;
      plotMgr.update(data, gcEnabled, timerange, forceNextUpdate);
      pendingUpdate = false;
      forceNextUpdate = false;
    }
  });
};

export const drawPlots = (force) => {
  pendingUpdate = true;
  if (force) {
    forceNextUpdate = true;
  }
  scheduleUpdate();
};

export const connect = () => {
  const uri = buildWebsocketURI();

  new WebSocketClient(
    uri,
    // onConfig
    (cfg) => {
      if (rafId !== null) cancelAnimationFrame(rafId);
      rafId = null;
      pendingUpdate = forceNextUpdate = false;
      plotMgr?.dispose();
      plotMgr = new PlotManager(cfg);
      statsMgr = new StatsManager(600, cfg);

      if (!navReady) {
        initNav(drawPlots);
        navReady = true;
      } else {
        updateVisibility();
      }
      const warning = document.getElementById("history-warning");
      if (warning) warning.hidden = true;
    },
    // onData
    (msg) => {
      statsMgr.pushData(msg);
      drawPlots(false);
    }
  );
};

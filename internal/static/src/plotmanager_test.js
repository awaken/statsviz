import assert from "node:assert/strict";
import test from "node:test";
import {loadModule} from "./vm_test_helper.js";

async function fixture(cost = 0, names = ["a", "b", "c"]) {
  let time=0, id=0, resize, disposed=0;
  const frames=new Map(), timers=new Map(), updates=[];
  class Plot {
    constructor(pd) {this.id=pd.name; this.visible=true; this.resized=0;}
    name() {return this.id;}
    createElement() {}
    isVisible() {return this.visible;}
    update(_range, _data, shapes) {updates.push({name:this.id, shapes}); time+=cost;}
    resize() {this.resized++;}
    dispose() {disposed++;}
  }
  const container={replaceChildren() {}, appendChild() {}};
  const {namespace}=await loadModule(new URL("./PlotManager.js",import.meta.url), {
    "./plot.js":{Plot,createVerticalLines: dates => dates.map(x0=>({x0}))},
  }, {
    document:{getElementById: name => name==="plots" ? container : {offsetWidth:600,offsetHeight:480}, createElement:()=>({})},
    window:{addEventListener(_event,fn) {resize=fn;},removeEventListener(_event,fn) {assert.equal(fn,resize);}},
    performance:{now:()=>time},
    requestAnimationFrame:fn=>(frames.set(++id,fn),id), cancelAnimationFrame:i=>frames.delete(i),
    setTimeout:fn=>(timers.set(++id,fn),id),clearTimeout:i=>timers.delete(i),
  });
  const manager=new namespace.default({series:names.map(name=>({name}))});
  return {manager,frames,timers,updates,resize:()=>resize(),disposed:()=>disposed};
}
const data={times:[1000],series:new Map(),events:new Map([["lastgc",[new Date(900)]]])};

test("audit: sustained updates do not starve later visible plots", async () => {
  const f=await fixture(8, ["a", "b", "c", "d", "e"]);
  for (const fn of f.frames.values()) fn(); f.frames.clear();
  for (let i=0;i<20;i++) {
    f.frames.set(`sample-${i}`,()=>f.manager.update(data,true,60));
    for (const [key,fn] of Array.from(f.frames)) {
      if (f.frames.delete(key)) fn();
    }
  }
  try {
    assert.ok(f.updates.some(u=>u.name==="e"), "last plot received no data while earlier plots updated 20 times");
  } finally {f.manager.dispose();}
});

test("plot manager caches events and releases frames, timers, and plots", async () => {
  const f=await fixture();
  for (const fn of f.frames.values()) fn(); f.frames.clear();
  assert.ok(f.manager.plots.every(p=>p.resized===1));
  f.manager.update({times:[]},true,60);
  f.manager.update(data,true,60);
  const shapes=f.updates[0].shapes.get("lastgc");
  f.manager.update(data,true,60);
  assert.equal(f.updates[3].shapes.get("lastgc"),shapes);
  f.manager.update({...data,events:new Map([["lastgc",[new Date(950)]]])},true,60);
  assert.notEqual(f.updates[6].shapes.get("lastgc"),shapes);
  f.manager.update(data,false,60);
  assert.equal(f.updates[9].shapes.size,0);
  f.manager.update(data,false,60);
  f.manager.plots[1].visible=false;
  f.resize(); f.resize(); assert.equal(f.timers.size,1);
  for (const fn of f.timers.values()) fn(); f.timers.clear();
  assert.equal(f.manager.plots[1].resized,1);
  f.resize(); f.manager.dispose(); f.manager.dispose();
  f.resize(); f.manager.update(data,true,60);
  assert.equal(f.frames.size,0); assert.equal(f.timers.size,0); assert.equal(f.disposed(),3);
});

import { readFile } from "node:fs/promises";
import vm from "node:vm";

export async function loadModule(url, stubs, globals = {}) {
  const context = vm.createContext({
    clearTimeout,
    console,
    setTimeout,
    ...globals,
  });
  const source = await readFile(url, "utf8");
  const entry = new vm.SourceTextModule(source, {
    context,
    identifier: url.href,
  });
  const modules = new Map();

  await entry.link(async (specifier) => {
    if (!Object.hasOwn(stubs, specifier)) {
      throw new Error(`unexpected import ${specifier}`);
    }
    if (modules.has(specifier)) return modules.get(specifier);

    const values = stubs[specifier];
    const module = new vm.SyntheticModule(
      Object.keys(values),
      function () {
        for (const [name, value] of Object.entries(values)) {
          this.setExport(name, value);
        }
      },
      { context, identifier: `stub:${specifier}` }
    );
    modules.set(specifier, module);
    return module;
  });
  await entry.evaluate();

  return { context, namespace: entry.namespace };
}

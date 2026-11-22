// Module resolution for the tests, so the tools run under plain `node --test`:
// - the sources import each other without an extension, as xmcp's compiler
//   resolves them; Node needs the `.ts`;
// - `xmcp/headers` only answers inside a request xmcp is serving; the tests
//   get a stand-in that returns the bearer they set.

const STAND_INS = {
    "xmcp/headers": new URL("./headers.mjs", import.meta.url).href,
};

export async function resolve(specifier, context, next) {
    if (STAND_INS[specifier]) {
        return { url: STAND_INS[specifier], shortCircuit: true };
    }

    const relative = specifier.startsWith("./") || specifier.startsWith("../");
    const fromSources = context.parentURL?.includes("/src/");
    if (relative && fromSources && !/\.[cm]?[jt]s$/.test(specifier)) {
        return next(`${specifier}.ts`, context);
    }

    return next(specifier, context);
}

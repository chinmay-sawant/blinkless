#!/usr/bin/env node
// WASM consumer check: build bindings/wasm, run it under node with the local
// Go wasm_exec.js, decode the versioned drawing-list JSON, read actual
// operations and referenced resources, and compare them with a native result
// for the same fixture. An operation count alone never passes.
//
// Usage: node scripts/wasm-consumer-check.mjs
//
// Requires node (v22+), the local Go toolchain, and testdata/wasm fixtures.

import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, statSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "..");

function fail(message) {
  console.error(`wasm-consumer-check: ${message}`);
  process.exit(1);
}

function check(condition, message) {
  if (!condition) {
    fail(message);
  }
}

function deepEqual(actual, expected, path) {
  if (actual === expected) {
    return;
  }

  if (typeof actual !== typeof expected || actual === null || expected === null) {
    fail(`${path}: ${JSON.stringify(actual)} != ${JSON.stringify(expected)}`);
  }

  if (Array.isArray(actual) !== Array.isArray(expected)) {
    fail(`${path}: array shape differs`);
  }

  if (Array.isArray(actual)) {
    if (actual.length !== expected.length) {
      fail(`${path}: length ${actual.length} != ${expected.length}`);
    }

    for (let i = 0; i < actual.length; i++) {
      deepEqual(actual[i], expected[i], `${path}[${i}]`);
    }

    return;
  }

  if (typeof actual === "object") {
    const keys = new Set([...Object.keys(actual), ...Object.keys(expected)]);

    for (const key of keys) {
      deepEqual(actual[key], expected[key], `${path}.${key}`);
    }

    return;
  }

  fail(`${path}: ${JSON.stringify(actual)} != ${JSON.stringify(expected)}`);
}

const workDir = mkdtempSync(join(tmpdir(), "blinkless-wasm-check-"));
process.on("exit", () => rmSync(workDir, { recursive: true, force: true }));

const manifestPath = join(repoRoot, "testdata", "wasm", "manifest.json");
const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
const fixturePath = join(repoRoot, "testdata", "wasm", manifest.fixture);
const html = readFileSync(fixturePath, "utf8");

// 1. Build the wasm artifact from the same package `make wasm` ships.
const wasmPath = join(workDir, "blinkless.wasm");
execFileSync("go", ["build", "-o", wasmPath, "./bindings/wasm"], {
  cwd: repoRoot,
  env: { ...process.env, GOOS: "js", GOARCH: "wasm" },
  stdio: ["ignore", "inherit", "inherit"],
});
check(statSync(wasmPath).size > 0, "wasm artifact is empty");

// 2. Native reference for the same fixture through the package's native entry
// point. This is the layout.DisplayList result the browser must agree with.
const nativePath = join(workDir, "native.json");
execFileSync(
  "go",
  [
    "run",
    "./bindings/wasm",
    "-fixture",
    fixturePath,
    "-width",
    String(manifest.request.width),
    "-height",
    String(manifest.request.height),
    "-out",
    nativePath,
  ],
  { cwd: repoRoot, stdio: ["ignore", "inherit", "inherit"] },
);
const native = JSON.parse(readFileSync(nativePath, "utf8"));

// 3. Load the runtime script from the same Go toolchain that built the wasm.
const goroot = execFileSync("go", ["env", "GOROOT"], { cwd: repoRoot }).toString().trim();
const wasmExecPath = join(goroot, "lib", "wasm", "wasm_exec.js");
vm.runInThisContext(readFileSync(wasmExecPath, "utf8"), { filename: wasmExecPath });

const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(readFileSync(wasmPath), go.importObject);
go.run(instance).catch((error) => {
  console.error("wasm-consumer-check: wasm runtime exited:", error);
  process.exit(1);
});

const deadline = Date.now() + 30000;
while (typeof globalThis.blinklessWASM !== "function") {
  if (Date.now() > deadline) {
    fail("blinklessWASM did not appear within 30 seconds");
  }

  await new Promise((resolveDelay) => setTimeout(resolveDelay, 10));
}

// 4. One real browser-shaped request.
const request = {
  html,
  mode: manifest.request.mode,
  width: manifest.request.width,
  height: manifest.request.height,
};
const response = globalThis.blinklessWASM(JSON.stringify(request));

check(response && response.ok === true, `request failed: ${JSON.stringify(response && response.error)}`);
check(response.mode === "display", `mode ${response.mode}, want display`);
check(response.mime === "application/json", `mime ${response.mime}, want application/json`);
check(response.schema === manifest.schema, `envelope schema ${response.schema}, want ${manifest.schema}`);
check(typeof response.version === "string" && response.version.length > 0, "envelope version missing");

const encoded = response.bytes;
check(encoded instanceof Uint8Array && encoded.byteLength > 0, "result bytes missing");
const payload = JSON.parse(Buffer.from(encoded.buffer, encoded.byteOffset, encoded.byteLength).toString("utf8"));

// 5. Structural contract: schema, units, counts, kinds, order, references.
check(payload.schema === manifest.schema, `payload schema ${payload.schema}, want ${manifest.schema}`);
check(payload.version === 1, `schema version ${payload.version}, want 1`);
check(payload.units === "points", `units ${payload.units}, want points`);
check(Number.isFinite(payload.pxPerPt) && Number.isFinite(payload.ptPerPx), "unit factors missing");
check(payload.width === response.width && payload.height === response.height, "canvas mismatch between envelope and payload");

const expected = manifest.expected;
check(Array.isArray(payload.ops) && payload.ops.length >= expected.minOps, `ops ${payload.ops.length}, want >= ${expected.minOps}`);
check(Array.isArray(payload.boxes) && payload.boxes.length >= expected.minBoxes, `boxes ${payload.boxes.length}, want >= ${expected.minBoxes}`);
check(Array.isArray(payload.fonts) && payload.fonts.length >= expected.minFonts, `fonts ${payload.fonts.length}, want >= ${expected.minFonts}`);
check(Array.isArray(payload.images) && payload.images.length >= expected.minImages, `images ${payload.images.length}, want >= ${expected.minImages}`);
check(Array.isArray(payload.groups), "groups must be an array");
check(Array.isArray(payload.order) && payload.order.length === payload.ops.length, "order must cover every operation");

const kinds = new Set(payload.ops.map((op) => op.kind));
for (const kind of expected.kinds) {
  check(kinds.has(kind), `decoded operations are missing kind ${kind}`);
}

const orderSeen = new Set();
for (const index of payload.order) {
  check(Number.isInteger(index) && index >= 0 && index < payload.ops.length, `order index ${index} out of range`);
  check(!orderSeen.has(index), `order repeats index ${index}`);
  orderSeen.add(index);
}

const fontIDs = new Set(payload.fonts.map((font) => font.id));
const imageIDs = new Set(payload.images.map((image) => image.id));
const groupIDs = new Set(payload.groups.map((group) => group.id));

for (const [index, op] of payload.ops.entries()) {
  check(Number.isFinite(op.x) && Number.isFinite(op.y) && Number.isFinite(op.w) && Number.isFinite(op.h), `op ${index} has non-finite geometry`);
  check(Number.isFinite(op.opacity), `op ${index} has non-finite opacity`);

  if (op.font) {
    check(fontIDs.has(op.font), `op ${index} references missing font ${op.font}`);
  }

  if (op.image) {
    check(imageIDs.has(op.image), `op ${index} references missing image ${op.image}`);
  }

  if (op.group !== undefined) {
    check(groupIDs.has(op.group), `op ${index} references missing group ${op.group}`);
  }

  if (op.kind === "text") {
    check(typeof op.text === "string" && op.text.length > 0, `text op ${index} has no text`);
    check(typeof op.font === "string" && op.font.length > 0, `text op ${index} has no font reference`);
  }

  if (op.kind === "image") {
    check(typeof op.image === "string" && op.image.length > 0, `image op ${index} has no payload reference`);
  }
}

for (const font of payload.fonts) {
  const data = Buffer.from(font.bytes, "base64");
  check(data.length === font.byteLength && font.byteLength > 0, `font ${font.id} bytes do not decode`);
  check(font.unitsPerEm > 0, `font ${font.id} has no unitsPerEm`);
}

for (const image of payload.images) {
  const data = Buffer.from(image.bytes, "base64");
  check(data.length === image.byteLength && image.byteLength > 0, `image ${image.id} bytes do not decode`);
  check(image.pixelWidth > 0 && image.pixelHeight > 0, `image ${image.id} has no pixel size`);
}

// 6. The browser result must agree with the native result operation by
// operation, not just in count.
deepEqual(payload, native, "payload");

console.log(
  `wasm-consumer-check: schema ${payload.schema}, ops ${payload.ops.length}, boxes ${payload.boxes.length}, ` +
    `fonts ${payload.fonts.length}, images ${payload.images.length}, groups ${payload.groups.length}`,
);
console.log("wasm-consumer-check: decoded operations and resources match the native result");

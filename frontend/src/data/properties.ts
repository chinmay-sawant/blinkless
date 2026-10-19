export interface ComparisonDimension {
  title: string;
  blink: string;
  blinkless: string;
  advantage: string;
}

export const ENGINE_STATS = {
  version: 'v0.0.2',
  modulePath: 'github.com/chinmay-sawant/blinkless',
  directDependencies: 2,
  language: 'Go',
  outputFormat: 'layout.DisplayList',
};

export const BLINK_COMPARISONS: ComparisonDimension[] = [
  {
    title: 'Language & Runtime',
    blink: 'Millions of lines of C++ with Oilpan garbage collection and platform toolchain requirements.',
    blinkless: 'Go. The default build is go build ./... with CGO_ENABLED=0.',
    advantage: 'A library call, not a browser process.',
  },
  {
    title: 'CGO Status',
    blink: 'Tightly coupled to platform C++ binaries. Complex cross-compilation matrix.',
    blinkless: 'The default build sets CGO_ENABLED=0. There is no C shared library target.',
    advantage: 'CGO stays off.',
  },
  {
    title: 'JavaScript Engine',
    blink: 'Deeply coupled with Google V8 (Ignition interpreter, TurboFan JIT, Web IDL bindings).',
    blinkless: 'Script is not executed.',
    advantage: 'No script runtime.',
  },
  {
    title: 'Process Architecture',
    blink: 'Multi-process architecture (Browser, Renderer, GPU) communicating over Mojo IPC.',
    blinkless: 'One library call in the caller\'s process.',
    advantage: 'No second process.',
  },
  {
    title: 'Memory Footprint',
    blink: 'A single renderer process typically consumes 50 to 100+ MiB RAM for V8 and Skia trees.',
    blinkless: 'A stylesheet over 32 MiB is rejected. Element nesting stops at 1024.',
    advantage: 'Hard size caps.',
  },
  {
    title: 'Output Model',
    blink: 'Direct GPU composition via Skia and cc compositor, managing shaders and tile caches.',
    blinkless: 'The result is layout.DisplayList. The host paints the pixels.',
    advantage: 'A drawing list, not pixels.',
  },
  {
    title: 'Direct Dependencies',
    blink: 'Hundreds of third-party submodules (V8, Skia, ICU, HarfBuzz, BoringSSL, libxml, libjpeg).',
    blinkless: 'Two direct modules: go-text/typesetting and tdewolff/canvas.',
    advantage: 'Two direct modules.',
  },
  {
    title: 'Security Attack Surface',
    blink: 'Frequent vector for CVEs (C++ use-after-free, V8 JIT escapes, IPC buffer overflows).',
    blinkless: 'Local file access is off by default. Restricted mode refuses private addresses.',
    advantage: 'File access off by default.',
  },
];

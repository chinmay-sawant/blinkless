# Native web view checklist

> **Status:** proposal. All implementation rows are open. This plan defines a native view built around Blinkless. It does not claim browser parity.
> **Scope decision:** Phase 0 chooses between bundled application pages and general websites. JavaScript is required for the second target.

## Goal

Build a native view that owns a live HTML document, accepts platform input, and presents Blinkless drawing lists on a native surface. Keep Blinkless as the HTML parser, CSS engine, layout engine, and source of paint operations. Add the browser behavior around that engine.

Blinkless currently runs `html.Parse`, `css.Apply`, and `layout.DisplayList`. The result is an ordered list of shapes, text, images, links, and element boxes. It does not start a browser or encode a page bitmap ([overview](../../../documentation/overview.md), [layout flow](../../../documentation/architecture/07-layout.md)). The public APIs are separate calls, and `css.Relayout` expects the parsed tree to remain unchanged ([library API](../../../documentation/library-api.md), [relayout contract](../../../css/relayout.go)).

The native renderer may use GPU textures or cached raster surfaces to present that list. Those surfaces are output caches. They do not replace the HTML tree, CSS state, layout boxes, or display list.

## Choose the first target

Complete this phase before committing to a public API. The two targets have different release requirements.

| Target | First release behavior | JavaScript |
|---|---|---|
| Bundled application pages | App-owned HTML and CSS, native input, links, forms needed by the app, scroll, and controlled resources | Can be deferred if all page behavior stays in Go |
| General websites | URL navigation, website resources, DOM-driven pages, browser input behavior, and a defined web compatibility level | Required |

Do not use the CSS catalog totals as a readiness score. The catalog measures a broad standards inventory, not compatibility with a particular application ([compatibility matrix](../../../documentation/compatibility-matrix.md)). Select real target pages and define expected behavior for each.

## Phase 0: define the contract

- [ ] List the first target pages and the HTML, CSS, input, navigation, and resource behavior they need.
- [ ] Choose the first target from the table above. Record whether JavaScript is in scope.
- [ ] Define the supported API: create and destroy a view, load HTML or a URL, set the CSS viewport and device scale, draw or request a frame, send input, and navigate backward or forward.
- [ ] Define document and view ownership. State which goroutine owns changes and how callers learn that a frame is ready.
- [ ] Define resource and script trust boundaries before accepting arbitrary URLs or page code.
- [ ] Capture Chromium results for the target pages and interactions. Record browser version, viewport, device scale, and input sequence.

**Gate:** a written target matrix names each supported behavior, its expected result, and its reference fixture. Keep all later phases tied to this matrix.

## Phase 1: add a live view controller

- [ ] Add a `View` controller above `html`, `css`, and `layout`. It owns the current document, base URL, viewport, interaction state, resource state, scroll positions, and current display list.
- [ ] Add explicit load, resize, input, invalidate, frame-ready, navigate, and close operations.
- [ ] Retain parsed HTML, stylesheets, font resources, and the last display list while their inputs remain valid.
- [ ] Route document changes through controller methods so each change triggers the required style, layout, and paint work.
- [ ] Cancel work and release resources when a document changes or the view closes.
- [ ] Define error and loading states for malformed HTML, failed resources, and interrupted navigation.

**Gate:** load a fixture, resize it, update one interaction state, and close it. Tests prove that the controller returns the expected display list and releases in-flight work.

## Phase 2: implement a live viewport and scrolling

- [ ] Add a viewport model with CSS-pixel dimensions, device scale, scroll position, and visible bounds.
- [ ] Add independent scroll positions for elements with scrollable overflow.
- [ ] Route wheel and touch deltas to the deepest eligible scroll container, then chain leftover movement to an ancestor.
- [ ] Add momentum, overscroll rules, scrollbars, and scroll anchoring for the chosen target level.
- [ ] Recompute sticky and fixed positioning when their scroll container moves.
- [ ] Reuse unchanged display operations and repaint newly exposed or invalidated regions.
- [ ] Keep viewport movement separate from document layout when only the scroll offset changes.
- [ ] Measure long documents and long chat threads. Add application-level row windowing only where the application owns the list data.

Blinkless currently has no interactive scroll viewport, scroll snapping, or scrollbar model ([scroll capability inventory](../../../documentation/compatibility-matrix.md)). Ownframe already has a window-level scroll path and an app-supplied row-window callback, but those do not provide general DOM scroll containers ([Ownframe window host](../../../../ownframe/internal/window/replay_viewport.go), [row-window hook](../../../../ownframe/internal/page/page_windowing.go)).

**Gate:** browser-reference fixtures cover slow drag, fast fling, nested scroll, edge chaining, resize while scrolled, sticky content, and content updates above the visible anchor. Record frame-time distributions on desktop and physical Android devices at supported refresh rates.

## Phase 3: connect input to HTML behavior

- [ ] Map hit-tested `layout.Box` IDs back to document nodes.
- [ ] Track pointer movement, pointer press, release, cancellation, and capture.
- [ ] Dispatch events with capture, target, and bubble phases. Support cancellation and default actions for the chosen target level.
- [ ] Update hover, active, and focus states through the view controller. Blinkless already accepts these states during CSS application ([CSS options](../../../css/css.go)).
- [ ] Implement keyboard focus order and keyboard activation for links and controls.
- [ ] Add the required text controls, selection, caret, clipboard, and IME behavior.
- [ ] Keep touch gesture ownership consistent when a gesture starts on a control and moves into a scroll container.

**Gate:** automated interaction tests cover taps, cancellation, keyboard focus, text replacement, selection, IME composition, and touch gestures. Compare visible state and resulting navigation with the reference browser.

## Phase 4: add navigation and resource lifecycle

- [ ] Connect link operations to URL resolution, same-document fragments, document replacement, and back/forward history.
- [ ] Extend the existing loader rather than creating a second independent loader. It already loads documents and document-relative stylesheets, images, and fonts, with URL policy and size limits ([loader architecture](../../../documentation/architecture/04-load.md)).
- [ ] Add asynchronous resource requests, cancellation, cache lifetime, revalidation, and progressive resource completion.
- [ ] Define URL, base URL, origin, redirect, cookie, and CORS behavior for the selected target.
- [ ] Apply security policy before requests reach the network or local file system.
- [ ] Expose loading, failure, reload, and navigation-completion events to the host.

**Gate:** tests cover relative and absolute resources, redirects, request cancellation, cache reuse, failed resources, fragment navigation, history, and denied local or network access.

## Phase 5: add JavaScript and a live DOM when required

This phase is required for general websites. Blinkless currently parses `<script>` content but never runs it ([script support status](../../../documentation/compatibility-matrix.md)). Its HTML nodes can be inspected and changed by Go code, but there is no browser DOM API or automatic invalidation contract ([HTML node](../../../internal/html/html.go)).

- [ ] Choose and isolate an ECMAScript runtime. Define its license, platform support, memory limits, and execution limits.
- [ ] Add `window` and `document` objects with a documented, versioned API subset.
- [ ] Add DOM queries and mutations, event listeners, and CSSOM changes.
- [ ] Connect mutations to style recalculation, layout, hit testing, and paint invalidation.
- [ ] Add script loading and execution order, tasks, microtasks, promises, timers, and animation-frame callbacks.
- [ ] Add required APIs such as location, history, and fetch only after origin and CORS rules are defined.
- [ ] Limit script CPU and memory use. Prevent one page from changing another page's state.

**Gate:** run a small script corpus in the target browser and Blinkless. Cover DOM creation, event handlers, asynchronous work, navigation, resource failures, and runtime limits. Publish unsupported APIs clearly.

## Phase 6: expose a native surface and lifecycle

- [ ] Define a renderer interface that accepts display-list updates and presents frames on a platform surface.
- [ ] Separate physical pixels, CSS viewport size, device scale, safe-area insets, and the keyboard-adjusted visual viewport.
- [ ] Define surface create, resize, pause, resume, loss, recreation, and destruction behavior.
- [ ] Schedule frames with the platform refresh cycle. Send damage regions so the renderer can reuse unchanged pixels.
- [ ] Define ownership and release rules for image payloads, font data, glyph caches, and GPU textures.
- [ ] Bridge native touch, pointer, keyboard, clipboard, and IME input to the view controller.
- [ ] Reuse Ownframe's Ebiten host where it fits. Keep platform input and window lifecycle out of Blinkless's parser and layout packages ([mobile binding](../../../../ownframe/run.go)).

**Gate:** attach, resize, pause, resume, and destroy a view on each supported platform. Test density changes, cutouts, keyboard appearance, rotation, and surface recreation on physical devices.

## Phase 7: expose accessibility

- [ ] Produce a semantic tree with roles, names, values, bounds, and supported actions.
- [ ] Keep semantic node identity stable across layout updates where possible.
- [ ] Map native accessibility actions back to focus, activation, text editing, and scroll operations.
- [ ] Announce dynamic content changes without repeating the whole page.

**Gate:** test reading order, labels, focus movement, activation, text entry, and scroll actions with TalkBack. Add equivalent checks for every other supported screen reader.

## Phase 8: set and enforce compatibility

- [ ] Maintain a target-page fixture suite with matching Chromium results.
- [ ] Track parser behavior, CSS behavior, geometry, paint output, interaction, and navigation as separate results.
- [ ] Add browser-driven tests for complete flows, not only parser or layout unit tests.
- [ ] Test resource limits, untrusted HTML, script limits, origin boundaries, and cancellation.
- [ ] Publish the supported HTML, CSS, JavaScript, and native-platform behavior for each release.

**Gate:** every supported target fixture passes its declared behavior checks. Record blocked behavior separately from passing behavior. Do not claim general Chrome compatibility from property counts or parser conformance alone.

## Release boundary

The bundled-page target can ship before JavaScript if all page behavior stays in the host application. It still needs a live view controller, scrolling, input, navigation appropriate to the app, resource handling, a native surface, and accessibility for supported controls.

The general-website target is not ready until JavaScript, DOM mutation, event handling, resource policy, navigation, and the declared compatibility suite pass together. Treat that as a separate release target, not a small extension to static layout.

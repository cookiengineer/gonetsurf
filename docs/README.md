# NetSurf Browser (`gonetsurf`) — Documentation Index

This directory is the authoritative implementation plan and progress ledger for the
`gonetsurf` project: a pure-Go web browser engine built on `golang.org/x/net/html` and
`golang.org/x` packages, with no third-party dependencies (in-tree forks are allowed and
tracked).

## Current state

Phases 0–3 are complete: the WHATWG URL layer, encoding sniffing, HTML microsyntaxes,
HTTP/1.1 + TLS + DNS + caches + cookies, and the HTML parser + `dom` all exist and are
tested (`go test ./...` green, 896/896 URL conformance cases). Phase 4 (CSS) is implemented:
the tokenizer, parser, grammar, selectors, values, colors, CSSOM, cascade, conditional
rules, media queries, properties, animations, and nesting all exist, and **Phase 4 is
complete** with all 14 tasks `Verified`. That includes the generated 353-property registry,
the stable WPT value corpus (2,520 passes, 0 failures, 260 documented skips), selectors
(0 failures, 78 tentative-feature skips), CSSOM, cascade/variables (`all`, `var()`,
cycles, `revert`), conditional rules, media queries, animations, and nesting. **Phase 5
(style resolution) is complete**: indexed selector matching, the UA stylesheet,
presentational hints, cascade/inheritance, custom properties, relative-unit and
per-property computed values, conditional integration, and incremental restyle are all
implemented and tested. Phase 6 (layout) has completed its layout-semantics pass:
box sizing constraints, block/inline/table/flex/grid formatting, floats, positioning,
logical properties, vertical flow, line-level bidi, generic-family text metrics and
kerning, scrollbar-aware overflow, and dirty-subtree reflow all exist. A 35-case
data-driven layout conformance harness passes, and `netsurfctl layout` exposes the full
parse → style → layout pipeline for diagnostics. Phase 6 is fully verified, including
sticky paint integration, real font metrics, and scrollbar painting. Phase 7 (painting) is nearly complete:
the `rendering` rasterizer is Verified (surfaces, paths, anti-aliasing, stroking,
clipping, Porter-Duff), backgrounds/borders/radii/shadows, PNG/JPEG/GIF/WebP image
decoding, Color 4 conversions (sRGB, Display P3, Lab/LCH, OKLab/OKLCH, and
wide-gamut color spaces), linear/radial/conic gradients, real font matching/
shaping/glyph caching, `@font-face`, SVG, stacking/opacity/2D+3D transforms with
individual transform properties, `srcset`/`picture`/video-poster replaced content,
and compositor/damage tracking are implemented; the headless PNG
renderer, golden pixel harness, and reftest comparator are implemented
(`netsurfctl render`, `netsurfctl reftest`), and a pure-Go X11 presenter opens
rendered pages (`netsurfctl show`). Phase 7 has 12 Verified tasks and 1 Partial
(OpenType feature/variation shaping, dependent on P10-TEXT-001). Phase 8 is
complete: X11 input, clipboard, MIT-SHM, tab/session and browsing-context model,
page scrolling, link/fragment/iframe navigation, omnibox search/autocomplete,
tab reordering, rasterized browser chrome, the broker-backed privacy dashboard,
and the `cmd/netsurf` interactive shell. External stylesheets and nested
`@import` rules are loaded through `adapters`, and CSS resource URLs are
resolved relative to their stylesheet. Real-site parity work also covers
multi-keyword `display`, `calc()` sizing, auto-margin centering, fixed/absolute
flex containers, clipped-overflow containment, nested scroll-container
hit-testing, generated `::before`/`::after` content, and resize-driven
media/layout updates. Scroll rendering uses
cached chrome, fonts, colors, glyph fallback, and offscreen-text culling.
Phase 9 is complete: IPC framing/shared memory, process supervision, sandbox
primitives, the network broker (cache/cookies + INET seccomp guard), and the
persistent storage broker are Verified. The sandboxed `netsurf` shell fetches
and stores everything through the broker. Totals: **97 Verified, 2 Partial,
18 Planned**. Phase 10 (JavaScript/Web APIs) and the cross-cutting backlog
remain. See [`handover.md`](./handover.md) for the full handover and
[`task-tracking.md`](./task-tracking.md) for the task-by-task ledger.

## Documents

| Document | Purpose |
|---|---|
| [`handover.md`](./handover.md) | **Start here.** Current state, limitations, and next steps. |
| [`architecture.md`](./architecture.md) | Process model, subsystem boundaries, interfaces, data flow. |
| [`implementation-plan.md`](./implementation-plan.md) | Phases 0–10 with features, tasks, deliverables, and acceptance criteria. |
| [`task-tracking.md`](./task-tracking.md) | Master task ledger with unit/E2E/spec/bug status for every task. |
| [`cache.md`](./cache.md) | Aggressive archive caching, archive banner, and the URL↔filesystem mapping. |
| [`privacy.md`](./privacy.md) | Network privacy, farbling, and the tracking-session cookie model. |
| [`security.md`](./security.md) | Sandboxing, trust boundaries, and the threat model. |
| [`conventions.md`](./conventions.md) | Code, naming, package, and test conventions. |
| [`specifications.md`](./specifications.md) | Specification inventory, provenance, and the missing-spec acquisition ledger. |

## Status legend (used in `task-tracking.md`)

### Implementation status

| Status | Meaning |
|---|---|
| `Planned` | Specified in this plan; no code written. |
| `WIP` | Partially implemented; not yet reliable. |
| `Implemented` | Code exists and builds; not necessarily verified. |
| `Verified` | Implemented, required tests are green, and spec conformance is assessed. |
| `Blocked` | Waiting on a dependency, spec, or decision. |
| `Dropped` | Intentionally not done; rationale recorded. |

### Verification columns

| Column | Values | Meaning |
|---|---|---|
| `UT` | `-` / `P` / `Y` | Unit tested: none / partial / yes. |
| `E2E` | `-` / `P` / `Y` | End-to-end tested: none / partial / yes. |
| `Spec` | `N/A`, `Unassessed`, `Partial`, `Compliant` | Specification compliance, with the governing spec and test suite recorded. |
| `Bugs` | `-` or issue links | Open defects known against the task. |

A task may only move to `Verified` when `UT=Y`, `E2E=Y`, `Spec=Compliant`, and `Bugs=-`.
`Spec=N/A` is permitted only for non-normative tooling tasks (e.g. build scripts).

### Task identifiers

Tasks are numbered by phase, e.g. `P4-CSS-012`. The full identifier is stable and is the
reference used in commit messages, tests, and bug reports.

## Governing rules

1. **Correctness first.** Behavior is driven by the specifications in `../specifications/`.
2. **No third-party runtime dependencies.** Standard library and `golang.org/x/*` only;
   forks live in-tree and are recorded against an upstream revision.
3. **Untrusted content is never trusted with the network or filesystem.** See `security.md`.
4. **The cache is a user asset.** It is human-readable, invertible, exportable, and
   reconstructible from files alone. See `cache.md`.
5. **Privacy is enforced across the whole pipeline**, not only in JavaScript. See `privacy.md`.

## How to update this plan

- New work starts as a `Planned` task in `implementation-plan.md`.
- Status and verification columns are updated in `task-tracking.md` as work lands.
- Any non-obvious decision is recorded in the relevant document under a "Decisions" heading
  rather than only in chat or commit messages.

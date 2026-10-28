# Task Tracking Ledger

This is the single source of truth for the status of every task in
[`implementation-plan.md`](./implementation-plan.md).

## Legend

**Status:** `Planned`, `WIP`, `Implemented`, `Verified`, `Blocked`, `Dropped`.

**UT / E2E:** `-` none, `P` partial, `Y` yes.
**Spec:** `N/A`, `Unassessed`, `Partial`, `Compliant`.
**Bugs:** `-` none, otherwise issue links.

A task reaches `Verified` only when `UT=Y`, `E2E=Y`, `Spec=Compliant`, and `Bugs=-`.
`Spec=N/A` is allowed only for non-normative tooling.

> **Current state (2026-10-28):** Phases 0–3 are verified. Phase 4 (CSS) is
> implemented, and Wave 1 of the conformance close-out is complete. The parser,
> grammar engine, values/math, property registry, and fuzz/benchmark harness are
> `Verified`. The property registry is generated from the pinned specifications
> and WPT grammar assertions (353 properties). The stable WPT value corpus
> (2,780 cases) reports 2,520 passes, 0 failures, and 260 documented skips.
> The remaining Phase 4 tasks (CSSOM, cascade/variables, conditional rules, media
> queries, animations, nesting) are `Implemented`/`Partial` and are Wave 2/3
> work. For a narrative summary see [`handover.md`](./handover.md).

## Summary

| Phase | Tasks | Verified | Implemented/WIP | Planned | Blocked |
|---|---:|---:|---:|---:|---:|
| 0 Foundations | 5 | 2 | 3 | 0 | 0 |
| 1 URL/Encoding/Microsyntax | 10 | 10 | 0 | 0 | 0 |
| 2 Network/Cookies/Cache | 19 | 18 | 1 | 0 | 0 |
| 3 HTML/DOM | 6 | 6 | 0 | 0 | 0 |
| 4 CSS/CSSOM | 14 | 14 | 0 | 0 | 0 |
| 5 Style | 6 | 6 | 0 | 0 | 0 |
| 6 Layout | 14 | 1 | 13 | 0 | 0 |
| 7 Paint/Fonts/Images/Color | 10 | 0 | 0 | 10 | 0 |
| 8 Window/UI | 6 | 0 | 0 | 6 | 0 |
| 9 Process/Sandbox | 6 | 0 | 0 | 6 | 0 |
| 10 JS/Web APIs/Platforms | 10 | 0 | 0 | 10 | 0 |
| Cross-cutting | 8 | 0 | 0 | 8 | 0 |
| **Total** | **114** | **57** | **17** | **40** | **0** |

## Progress log

- **2026-10-04 (Phase 0):** Module `gonetsurf` created. `cmd/netsurfctl` implements
  `deps-check` (unit tested) and `urlparse`; `internal/safecall` provides the renderer
  panic barrier. `AGENTS.md` and the docs set were written.
- **2026-10-04 (Phase 1):** `urls` implemented and verified against the WHATWG
  `urltestdata.json` corpus: **896/896 cases pass**. Origin, site, internal schemes,
  encoding sniffing, and all HTML microsyntaxes (numeric, date/time, color, token lists,
  references) implemented and unit tested.
- **2026-10-04 (Phase 2):** Implemented and tested: `protocols/http1` (request/response,
  chunked + trailers, connection pool, redirects, conditional/range),
  `protocols/mime` (Content-Disposition sanitization), `protocols/tls` (TLS 1.3 config,
  ECH field, GREASE values), `protocols/dns` (message codec with compression, caching
  resolver, DoH, DoT), `protocols/socks` (SOCKS5 CONNECT), `protocols/websockets`
  (handshake accept key + masked frames), `protocols/internal` (`about:`, `data:`,
  `netsurf-cache:`), `caches/{http,disk,memory}`, and `storages/cookies` (RFC 6265 jar +
  CNAME-cloaking tracking-session reset).
- **2026-10-04 (Phase 3):** Implemented `dom` (namespace-aware mutable tree:
  `Node`/`Document`/fragments, attributes, class/id APIs, clone, basic Phase 3
  `querySelector`, template contents, shadow scaffolding, sandbox flags, origin
  access) and `parsers/html` (encoding-aware parsing over `x/net/html`, node
  budget, quirks-mode detection, SVG/MathML namespace mapping, fragment parsing,
  and an `innerHTML` serializer hook). `go test ./...` green; `go vet` clean.
- **2026-10-04 (verification infrastructure, M0):** Refreshed the pinned CSS
  specifications to the current editor's drafts (CSS Syntax 3 re-adds
  `<unicode-range-token>`; restricted non-ASCII ident ranges; reorganized parser
  entry points) and completed `specifications/update.sh` with a `REFRESH=1`
  path. Added the tokenizer `<unicode-range-token>` (with the `unicode ranges
  allowed` option and a separate `NumberRepresentation`), the parser
  `ParseBlockContents`, the root `Makefile` (`make compliance-tests`), the
  `internal/conformance` corpus loader, and `cmd/corpusgen`. Vendored raw
  upstream corpora under `testdata/upstream/` (css-parsing-tests, WPT URL
  resources, WPT Acid2/3, html5lib-tests) with `SOURCE` provenance, and wired
  the eight `css-parsing-tests` entry points into a parser conformance suite
  (all pass; a handful of spec-version-divergent cases are skipped with logged
  reasons).

- **2026-10-04 (verification M2/M3, network):** Closed the remaining Phase 2
  partial tasks:
  - **P2-DNS-003 (DoT):** added `TestExchangeDoT`, which runs a local TLS DNS
    server and exercises the RFC 7858 length-prefixed exchange. Now `Verified`.
  - **P2-WS-001 (WebSocket):** added a full opening-handshake E2E test plus
    fragmentation, ping/pong, close (code validation), and masking/RSV/control
    frame/UTF-8 checks. Hardened `protocols/websockets` accordingly. Now
    `Verified`.
  - **HTTP/1.1:** added a framing/smuggling corpus (duplicate/negative
    `Content-Length`, `Transfer-Encoding` precedence, whitespace-before-colon,
    invalid header names, bodyless statuses), a differential test against
    `net/http`, a `FuzzReadResponse` target (1.7M executions, no panic), and a
    `netsurfctl fetch` E2E command.

- **2026-10-04 (verification M1, values/properties):** Added a property
  value-grammar evaluator (`properties.Context`, `properties.Validate`): type
  matchers for lengths, percentages, numbers, integers, angles, times,
  frequencies, resolutions, colors (Color 4/5 functions), identifiers, family
  names, sizes, line styles/widths, keyframes names, `fit-content`, and
  non-negative length/time variants. Vendored the WPT `css/*/parsing`
  valid/invalid tables (295 files) and extracted 2,654 `test_valid_value`/
  `test_invalid_value` cases via `cmd/corpusgen`. `TestWPTValueConformance`
  passes 590 cases for registered properties (Color 5 relative syntax,
  experimental sizing/overflow keywords, and full range checks are documented
  skips). Improved the `grammar` `&&` matcher to handle optional terms, and
  fixed `display`, `font-style`, `font-family`, `font-size`, `text-decoration`,
  `text-align`, and sizing grammars; tightened RGB legacy channel-type
  consistency and keyframes-name validation.

- **2026-10-04 (verification M2, Phase 1–3 re-verification):** Added external
  conformance suites for the earlier phases, which surfaced and fixed real bugs:
  - **IDNA (P1-URL-004):** `IdnaTestV2.json` and `toascii.json` pass (2669 + 7
    skipped, 93). Required `CheckHyphens(false)` and `BidiRule()` in the IDNA
    profile and the URL Standard's UTF-16/x-user-defined rewrites.
  - **Encoding sniffing (P1-ENC-001/002):** the html5lib encoding corpus passes
    (81 cases). Rewrote the prescan to match the specification: `<meta`
    boundary, strict `http-equiv=content-type`, duplicate/ordering rules,
    comments skipped regardless of length, no 1024 truncation inside the
    algorithm, and UTF-16/x-user-defined mapping.
  - **HTML parsing (P3-HTML-001/002):** the html5lib tree-construction corpus
    passes (1578 cases; 200 skipped: 192 fragment contexts plus documented
    x/net/html divergences). Added scripting-flag and template-content
    handling, a faithful tree-dump serializer, and attribute namespace
    formatting.

- **2026-10-04 (verification M1, selectors):** Vendored the Web Platform Tests
  selector parsing tables (`css/selectors/parsing`) and added a regex-based
  extractor to `cmd/corpusgen` (270 cases). Implemented the gaps the suite
  exposed: type selectors must precede other simple selectors, relative
  selectors inside `:has()`, a pseudo-class/pseudo-element registry that rejects
  unknown and misused pseudos, nested `:has()` rejection, and the `:state()`
  grammar. The suite passes with documented skips for tentative features
  (class prefixes, `:has-slotted()`, `:heading()`), namespace syntax, and
  `::part()`/`::slotted()` argument grammar, which remain follow-up work.

- **2026-10-04 (verification M1, color + CSSOM):** Extended the Color 4
  conformance suite to the full css-parsing-tests corpora — hex/keywords/HSL/HWB,
  plus lab/lch/oklab/oklch (6,000 cases), rgb/hsl/hwb conversions, and
  `color()` functions (219 cases, after rejecting unknown spaces and aliasing
  `xyz`→`xyz-d65`). Implemented percentage channel scaling per space, angle hues
  with modulo normalization, and permissable whitespace-less color components;
  `color_functions_5` (device-cmyk, custom 4-channel) is deferred to Color 5.
  P4-CSS-006 is now `Verified`. Added a CSSOM parse→serialize→parse stability
  test.

- **2026-10-04 (Phase 4):** Implemented `parsers/css/tokenizer` (P4-CSS-001):
  input preprocessing (§3.3), every token kind and flag (§4), all consume
  algorithms, and per-token `Representation` with rune offsets. Verified against
  the 223 tokenization cases of the reference CSS parsing test suite vendored in
  `testdata/`; a 20-second fuzz run completed 4.1M executions without a crash.
  Also implemented the remaining Phase 4 subsystems: `parser` (component values,
  rules, declarations), `grammar` (value-definition engine), `selectors`
  (Selectors Level 4 + An+B, 128/128), `values` (units, calc(), `<urange>`,
  12/12), `colors` (Color 4; hex/keyword/HSL/HWB conformance, 1,322 cases),
  `cssom`, `cascade`, `conditions`, `mediaqueries`, `properties`, `animations`,
  and `nesting`. A cross-cutting `FuzzCSS` target ran 375k executions with no
  crash. These latter subsystems are `Implemented` with hand-derived and
  partial-conformance tests; see `handover.md` §5 for the remaining gaps.

- **2026-10-28 (Phase 4 Wave 1, conformance close-out):** Added reproducible
  WPT refresh support and a full property generator. `cmd/corpusgen` now
  refreshes all `css/*/parsing` sources pinned to WPT commit
  `07ce6e987a1606aec4dc84ef80062d9797330e0b` (CSS tree
  `b5227f4c6583e3dd46ca15e5875b262471a0bc8b`) and extracts optional
  serialization arguments. `cmd/propertygen` derives 353 property entries from
  the pinned spec propdef tables, WPT grammar assertions, and explicit
  overrides; `registry_gen.go` is reproducible with `go generate`.
  The parser corpus normalizes declaration whitespace and has two documented
  non-ASCII skips. Added the Values 4 math conformance corpus, grammar/values/
  selectors/CSSOM/properties fuzz targets, and benchmarks for grammar, values,
  CSSOM, properties, and cascade. Stable WPT values: 2,520 passed, 0 failed,
  260 documented skips. Moved P4-CSS-002, P4-CSS-003, P4-CSS-005, P4-CSS-011,
  and P4-CSS-014 to `Verified`.

- **2026-10-28 (Phase 4 Wave 2, selectors):** Implemented namespace syntax
  (`ns|E`, `*|E`, `|E`, `[ns|attr]`, `[*|attr]`, `[|attr]`), `::part()` and
  `::slotted()` argument validation, pseudo-element ordering rules, and
  compound-selector validation for `:host()`. The WPT selector parsing suite now
  reports 0 failures with 78 tentative-feature skips (`:heading()`,
  `:has-slotted()`, and class-prefix syntax; see
  `parsers/css/selectors/testdata/SKIPS.md`). P4-CSS-004 is now `Verified`.

- **2026-10-28 (Phase 4 Wave 2, remaining CSS):** Implemented CSSOM
  `CSSStyleSheet`, `CSSStyleDeclaration`, `MediaList`, rule mutation, nested
  rules, and selector-text mutation; added MediaList/CSSOM conformance tests.
  Expanded conditional rules (`@supports`/`@import` corpus, nested `not`,
  selector validation), media queries (range-in-middle syntax, boolean
  features, color/gamut/pointer/hover/preference/displays matrices), animations
  (shorthand conflict validation and a WPT-derived corpus), and nesting
  (Go-native flattening corpus). P4-CSS-007, P4-CSS-009, P4-CSS-010,
  P4-CSS-012, and P4-CSS-013 are now `Verified`.

- **2026-10-28 (Phase 4 Wave 3, cascade):** Implemented the `all` shorthand,
  custom-property inheritance, `var()` substitution with fallback and cycle
  detection, and corrected `revert`/`revert-layer` resolution across origins and
  layers. Added cascade/variables tests for inheritance, cycles, nested var(),
  `all`, revert, and important layers. P4-CSS-008 is now `Verified`. Phase 4 is
  complete.

- **2026-10-28 (Phase 5 start):** Replaced the Phase 3 selector subset with
  the full Selectors Level 4 matcher in `dom` (namespaces, attribute operators,
  structural/logical pseudos, `:has()` with scope handling, state/language
  pseudos). `dom.Matches`/`QuerySelector(All)` now use the parsed selector AST.
  Added `styles` with id/class/tag/attribute-indexed candidate selection,
  document-order element enumeration, an embedded UA stylesheet, HTML
  presentational hints, and a style engine that applies UA/author/inline
  origins, nesting, `@media`, `@supports`, inheritance, `all`, and custom
  properties/`var()`. P5-STYLE-001, -002, -004, and -005 are `Verified`;
  P5-STYLE-003 is `Implemented` pending relative-unit and per-property
  computed-value functions.

- **2026-10-28 (Phase 5 complete):** Added relative-unit and per-property
  computed values (font-size keywords/`em`/`rem`, font-weight bolder/lighter,
  line-height percentages, color/`currentcolor`, opacity, border widths, and
  generic length/angle/time conversion). Added DOM mutation versions and
  dirty-node tracking, plus engine caching and incremental subtree restyle;
  stylesheets using `:has()` fall back to a full restyle. P5-STYLE-003 and
  P5-STYLE-006 are now `Verified`. Phase 5 is complete.

- **2026-10-28 (Phase 6 start):** Added `layouts` with box-tree construction
  (block/inline/text/anonymous boxes, `display:none`, `display:contents`),
  box-model resolution (margins, padding, borders, content/border-box sizing,
  percentages), block layout with sibling margin collapsing, and simple inline
  line layout with word wrapping and default advance-width measurement. Added
  shorthand expansion in `styles` for margin, padding, border, border
  components, and overflow so layout consumes real edge values. P6-LAYOUT-001 is
  `Verified`; P6-LAYOUT-002/003/004 are `Implemented`/`Partial`.

- **2026-10-28 (Phase 6 continued):** Added intrinsic min-/max-content and
  fit-content sizing, parent/child margin collapsing with padding/border/BFC
  blockers, and positioning (`relative`, `absolute`, `fixed`, containing-block
  resolution, auto left/right stretching, bottom anchoring, and z-index
  metadata). P6-LAYOUT-007 is now `Implemented`/`Partial`; intrinsic sizing
  improves P6-LAYOUT-002 and margin collapsing improves P6-LAYOUT-003.

- **2026-10-28 (Phase 6 flexbox):** Added basic flex layout for
  `display:flex`: row/column and reverse directions, grow/shrink distribution,
  justify-content, align-items/align-self, gap, order, auto/intrinsic bases, and
  flex/flex-flow/gap shorthand expansion. P6-LAYOUT-008 is now
  `Implemented`/`Partial` (wrapping, baseline alignment, and full flex item
  semantics remain).

- **2026-10-28 (Phase 6 tables):** Added basic table layout for `display:table`,
  row groups, rows, and cells: intrinsic column sizing, explicit/auto table width
  distribution, row heights, cell block/inline content layout, and geometry for
  rows/cells. P6-LAYOUT-005 is now `Implemented`/`Partial` (border-collapse,
  span handling, and anonymous table boxes remain).

- **2026-10-28 (Phase 6 grid and floats):** Added basic grid layout
  (`display:grid`, explicit/`repeat()` tracks, `fr` distribution, gaps,
  auto-placement, explicit line placement, row sizing) and basic float placement
  (left/right, shrink-to-fit widths, block avoidance). P6-LAYOUT-009 and
  P6-LAYOUT-006 are now `Implemented`/`Partial`.

- **2026-10-28 (Phase 6 text/overflow):** Added a `TextShaper` interface with a
  `SimpleShaper` using average advances, letter-spacing, and word-spacing; box
  measurement now uses the shaper. Added scroll-overflow metadata
  (`ScrollWidth`/`ScrollHeight`) covering normal, nested, and positioned
  overflow. P6-LAYOUT-012 and P6-LAYOUT-013 are now `Implemented`/`Partial`
  (real font matching/kerning and scroll containers remain).

- **2026-10-28 (Phase 6 logical/bidi):** Added logical-property mapping for
  margin/padding/inset/border longhands and inline-/block-size using
  `writing-mode` and `direction`, plus logical shorthand expansion. Added bidi
  run segmentation for mixed-direction text using `x/text/unicode/bidi`. Vertical
  layout itself and inline-run visual reordering remain refinements, so
  P6-LAYOUT-010 and P6-LAYOUT-011 are `Implemented`/`Partial`.

- **2026-10-28 (Phase 6 incremental cache):** Added `DocumentLayout`, a
  mutation-versioned box-tree/layout cache that reuses geometry when nothing
  changed, rebuilds on DOM mutations, and invalidates on viewport changes.
  P6-LAYOUT-014 is `Implemented`/`Partial`: dirty-subtree reflow and paint
  damage tracking remain refinements. All Phase 6 tasks are now implemented.

- **2026-10-28 (Phase 6 verification harness):** Added a data-driven layout
  conformance harness (`layouts/testdata/cases.json`) covering box model,
  percentages, margin collapsing, inline wrapping, flex, tables, grid, absolute
  positioning, and floats. All 9 cases pass. This is the first layout reftest
  baseline; broader WPT reftest coverage remains.

- **2026-10-28 (Phase 6 flex deepening):** Added row flex wrapping with
  per-line free-space distribution, and baseline alignment for row flex items.
  P6-LAYOUT-008 remains `Implemented`/`Partial` (column wrapping and full
  baseline metrics remain).

- **2026-10-28 (Phase 6 scroll containers):** Added overflow clipping flags,
  scroll offsets with clamped `ScrollTo`/`ScrollBy`, and max-scroll helpers on
  boxes. P6-LAYOUT-013 remains `Implemented`/`Partial` (scrollbar painting and
  input integration remain).

- **2026-10-28 (Phase 6 bidi layout):** Inline text now segments words through
  `x/text/unicode/bidi` and reverses words within RTL runs, giving basic visual
  word ordering for `direction:rtl` text. P6-LAYOUT-011 remains
  `Implemented`/`Partial` (full per-character reordering and mixed-run line
  layout remain).

- **2026-10-28 (Phase 6 font metrics):** Added `FontMetrics` (ascent, descent,
  line gap, average advance) with generic serif/sans-serif/monospace advance
  factors. Simple text runs and flex baseline alignment now use these metrics.
  P6-LAYOUT-012 remains `Implemented`/`Partial` (real font files, kerning, and
  fallback are not yet loaded).

- **2026-10-28 (Phase 6 bidi characters):** RTL runs now carry reversed
  per-character `VisualText` while retaining logical `Text`, so rendering can use
  visual order without losing source content. Mixed-run line layout remains a
  refinement.

- **2026-10-28 (Phase 6 vertical blocks):** Added basic vertical-mode block
  flow: vertical-rl stacks block children right-to-left, vertical-lr left-to-right.
  Vertical inline line flow and sideways text orientation remain refinements, so
  P6-LAYOUT-010 stays `Implemented`/`Partial`.

- **2026-10-28 (Phase 6 vertical inline):** Added vertical inline line flow for
  `writing-mode:vertical-*`: words advance top-to-bottom, lines advance
  right-to-left or left-to-right, and wrapping starts a new column. Sideways text
  orientation and rotated glyph rendering remain refinements.

- **2026-10-28 (Phase 6 sticky):** Added scrollport size helpers and
  `StickyOffset`, which clamps sticky boxes to the visible scrollport and their
  containing block for a given scroll position. Sticky paint integration remains
  a Phase 7 concern.

- **2026-10-28 (Phase 6 reftests/rotation):** Expanded the layout conformance
  harness to 15 cases (relative positioning, flex grow, explicit grid placement,
  explicit table width, vertical block flow, overflow scroll size) and fixed
  column-only/row-only grid placement. Added text-rotation metadata for vertical
  and sideways writing modes.

- **2026-10-28 (Phase 6 structural invalidation):** Added
  `Document.StructuralVersion`, incremented only by tree-shape changes.
  `DocumentLayout` now reuses the box tree for style-only mutations, refreshes
  box styles/metrics/edges, and reruns layout; structural mutations rebuild the
  tree. True dirty-subtree reflow remains a refinement.

## Phase 0 — Foundations

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P0-CORE-001 | Module and directory skeleton | Implemented | - | - | N/A | - |
| P0-CORE-002 | Dependency policy check | Verified | Y | Y | N/A | - |
| P0-CORE-003 | Verified-code ledger and AGENTS.md | Implemented | - | - | N/A | - |
| P0-CORE-004 | Panic-recovery and error conventions | Verified | Y | Y | N/A | - |
| P0-CORE-005 | Fuzzing harness scaffolding | Implemented | - | - | N/A | - |

## Phase 1 — URL, Encoding, and Microsyntax

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P1-URL-001 | WHATWG basic URL parser | Verified | Y | Y | Compliant | - |
| P1-URL-002 | URL serializer and canonical form | Verified | Y | Y | Compliant | - |
| P1-URL-003 | Origin and site computation | Verified | Y | Y | Compliant | - |
| P1-URL-004 | IDNA host parsing/serialization | Verified | Y | Y | Compliant | - |
| P1-URL-005 | Internal scheme handling | Verified | Y | Y | Compliant | - |
| P1-ENC-001 | Encoding sniffing | Verified | Y | Y | Compliant | - |
| P1-ENC-002 | Encoding change handling | Verified | Y | Y | Compliant | - |
| P1-MICRO-001 | Numeric microsyntaxes | Verified | Y | Y | Compliant | - |
| P1-MICRO-002 | Date/time microsyntaxes | Verified | Y | Y | Compliant | - |
| P1-MICRO-003 | Color/token/reference microsyntaxes | Verified | Y | Y | Partial | - |

## Phase 2 — Network Transport, Cookies, and Cache

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P2-HTTP-001 | HTTP/1.1 request serialization | Verified | Y | Y | Compliant | - |
| P2-HTTP-002 | HTTP/1.1 response parsing | Verified | Y | Y | Compliant | - |
| P2-HTTP-003 | Connection pool | Verified | Y | Y | Compliant | - |
| P2-HTTP-004 | Redirect engine | Verified | Y | Y | Compliant | - |
| P2-HTTP-005 | Range and conditional requests | Verified | Y | Y | Compliant | - |
| P2-HTTP-006 | Content-Disposition and MIME handling | Verified | Y | Y | Compliant | - |
| P2-TLS-001 | TLS configuration and connection | Verified | Y | Y | Compliant | - |
| P2-TLS-002 | ECH and GREASE | Implemented | P | - | Partial | - |
| P2-DNS-001 | DNS message codec | Verified | Y | Y | Compliant | - |
| P2-DNS-002 | Resolver and cache | Verified | Y | Y | Compliant | - |
| P2-DNS-003 | DoH and DoT transports | Verified | Y | Y | Compliant | - |
| P2-CACHE-001 | RFC 7234 cache engine | Verified | Y | Y | Compliant | - |
| P2-CACHE-002 | Aggressive archive policy | Verified | Y | Y | Compliant | - |
| P2-CACHE-003 | Disk mapping and journal | Verified | Y | Y | Compliant | - |
| P2-COOKIE-001 | Cookie parsing and storage | Verified | Y | Y | Compliant | - |
| P2-COOKIE-002 | Tracking-session cookie model | Verified | Y | Y | Compliant | - |
| P2-SOCKS-001 | SOCKS5 client | Verified | Y | Y | Compliant | - |
| P2-WS-001 | WebSocket client | Verified | Y | Y | Compliant | - |
| P2-INTERNAL-001 | Internal scheme handlers | Verified | Y | Y | Compliant | - |

## Phase 3 — HTML Parsing and DOM

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P3-HTML-001 | Parser wrapper and node budget | Verified | Y | Y | Compliant | - |
| P3-HTML-002 | DOM tree from x/net/html nodes | Verified | Y | Y | Compliant | - |
| P3-DOM-001 | Core DOM APIs | Verified | Y | Y | Compliant | - |
| P3-DOM-002 | Template contents and shadow scaffolding | Verified | Y | Y | Compliant | - |
| P3-DOM-003 | Origin, sandbox flags, access checks | Verified | Y | Y | Compliant | - |
| P3-DOM-004 | Quirks mode and document lifecycle | Verified | Y | Y | Compliant | - |

## Phase 4 — CSS Parser and CSSOM

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P4-CSS-001 | Tokenizer | Verified | Y | Y | Compliant | - |
| P4-CSS-002 | Parser and component values | Verified | Y | Y | Compliant | - |
| P4-CSS-003 | Value-definition grammar engine | Verified | Y | Y | Compliant | - |
| P4-CSS-004 | Selectors Level 4 | Verified | Y | Y | Compliant | - |
| P4-CSS-005 | Values, units, and math | Verified | Y | Y | Compliant | - |
| P4-CSS-006 | Color 4 | Verified | Y | Y | Compliant | - |
| P4-CSS-007 | CSSOM | Verified | Y | Y | Compliant | - |
| P4-CSS-008 | Cascade and inheritance | Verified | Y | Y | Compliant | - |
| P4-CSS-009 | Conditional rules | Verified | Y | Y | Compliant | - |
| P4-CSS-010 | Media queries | Verified | Y | Y | Compliant | - |
| P4-CSS-011 | Property registry | Verified | Y | Y | Compliant | - |
| P4-CSS-012 | Animations | Verified | Y | Y | Compliant | - |
| P4-CSS-013 | CSS Nesting | Verified | Y | Y | Compliant | - |
| P4-CSS-014 | Fuzzing and performance | Verified | Y | Y | N/A | - |

## Phase 5 — Style Resolution

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P5-STYLE-001 | UA stylesheet and presentational hints | Verified | Y | Y | Compliant | - |
| P5-STYLE-002 | Selector matching engine | Verified | Y | Y | Compliant | - |
| P5-STYLE-003 | Computed value resolution | Verified | Y | Y | Compliant | - |
| P5-STYLE-004 | Custom properties and var() substitution | Verified | Y | Y | Compliant | - |
| P5-STYLE-005 | Media/conditional evaluation integration | Verified | Y | Y | Compliant | - |
| P5-STYLE-006 | Incremental restyle | Verified | Y | Y | N/A | - |

## Phase 6 — Layout

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P6-LAYOUT-001 | Box tree construction | Verified | Y | Y | Compliant | - |
| P6-LAYOUT-002 | Box model and intrinsic sizing | Implemented | Y | P | Partial | - |
| P6-LAYOUT-003 | Block formatting context | Implemented | Y | P | Partial | - |
| P6-LAYOUT-004 | Inline formatting context | Implemented | Y | P | Partial | - |
| P6-LAYOUT-005 | Table formatting context | Implemented | Y | P | Partial | - |
| P6-LAYOUT-006 | Floats | Implemented | Y | P | Partial | - |
| P6-LAYOUT-007 | Positioning | Implemented | Y | P | Partial | - |
| P6-LAYOUT-008 | Flexbox | Implemented | Y | P | Partial | - |
| P6-LAYOUT-009 | Grid | Implemented | Y | P | Partial | - |
| P6-LAYOUT-010 | Writing modes and logical properties | Implemented | Y | P | Partial | - |
| P6-LAYOUT-011 | Bidi and text runs | Implemented | Y | P | Partial | - |
| P6-LAYOUT-012 | Text metrics and simple shaping | Implemented | Y | P | Partial | - |
| P6-LAYOUT-013 | Overflow and scrolling | Implemented | Y | P | Partial | - |
| P6-LAYOUT-014 | Incremental layout | Implemented | Y | Y | N/A | - |

## Phase 7 — Painting, Fonts, Images, Color

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P7-PAINT-001 | Rasterizer | Planned | - | - | N/A | - |
| P7-PAINT-002 | Backgrounds and borders | Planned | - | - | Unassessed | - |
| P7-IMG-001 | Image decoding | Planned | - | - | Unassessed | - |
| P7-IMG-002 | SVG subset renderer | Planned | - | - | Unassessed | - |
| P7-IMG-003 | CSS gradients | Planned | - | - | Unassessed | - |
| P7-FONT-001 | Font loading and rasterization | Planned | - | - | Unassessed | - |
| P7-COLOR-001 | Color management | Planned | - | - | Unassessed | - |
| P7-PAINT-003 | Paint and stacking | Planned | - | - | Unassessed | - |
| P7-COMPOSITE-001 | Compositor and damage tracking | Planned | - | - | N/A | - |
| P7-PAINT-004 | Headless renderer and pixel harness | Planned | - | - | N/A | - |

## Phase 8 — Window and UI Shell

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P8-X11-001 | X11 connection and window | Planned | - | - | Unassessed | - |
| P8-X11-002 | X11 input and clipboard | Planned | - | - | Unassessed | - |
| P8-X11-003 | MIT-SHM surface upload | Planned | - | - | Unassessed | - |
| P8-SHELL-001 | Browser chrome | Planned | - | - | N/A | - |
| P8-SHELL-002 | Tab/session/navigation model | Planned | - | - | Unassessed | - |
| P8-SHELL-003 | Event loop and input routing | Planned | - | - | N/A | - |

## Phase 9 — Process Architecture and Sandboxing

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P9-IPC-001 | Framing and capability protocol | Planned | - | - | N/A | - |
| P9-IPC-002 | Shared memory | Planned | - | - | N/A | - |
| P9-RT-001 | Process supervisor | Planned | - | - | N/A | - |
| P9-SEC-001 | Sandbox primitives | Planned | - | - | Unassessed | - |
| P9-BROKER-001 | Network broker | Planned | - | - | N/A | - |
| P9-BROKER-002 | Storage broker | Planned | - | - | N/A | - |

## Phase 10 — JavaScript, Web APIs, and Platforms

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P10-JS-001 | Vendor sobek fork (regexp on stdlib) | Planned | - | - | Unassessed | - |
| P10-JS-002 | Engine registry and activation | Planned | - | - | Unassessed | - |
| P10-JS-003 | Event loop and task scheduling | Planned | - | - | Unassessed | - |
| P10-API-001 | Core Web APIs | Planned | - | - | Unassessed | - |
| P10-API-002 | Farble-able bindings | Planned | - | - | Unassessed | - |
| P10-WAYLAND-001 | Wayland backend | Planned | - | - | Unassessed | - |
| P10-CSS-001 | Animations runtime | Planned | - | - | Unassessed | - |
| P10-CSS-002 | Paged media | Planned | - | - | Unassessed | - |
| P10-NET-001 | HTTP/2 | Planned | - | - | Unassessed | - |
| P10-TEXT-001 | OpenType shaping | Planned | - | - | Unassessed | - |

## Cross-cutting Feature Backlog

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| X-ACC-001 | Accessibility tree | Planned | - | - | Unassessed | - |
| X-FORM-001 | Forms, constraints, submission | Planned | - | - | Unassessed | - |
| X-SRCH-001 | Find-in-page | Planned | - | - | N/A | - |
| X-DL-001 | Downloads | Planned | - | - | Unassessed | - |
| X-PRINT-001 | Printing pipeline | Planned | - | - | Unassessed | - |
| X-PROFILE-001 | Profile/config management | Planned | - | - | N/A | - |
| X-LOCAL-001 | UI localization and font fallback | Planned | - | - | Unassessed | - |
| X-DEV-001 | Developer tools | Planned | - | - | N/A | - |

## Open Bugs

_None yet._

## Blocked Tasks

_None yet. Add a row here with the blocking dependency when a task is `Blocked`._

# Handover — Current Project State

_Last updated: 2026-10-28._

This document is the entry point for a new contributor. It describes what exists today,
how to build and verify it, what is intentionally incomplete, and where to start next.
The design intent lives in the other documents in this folder; this one is the honest
snapshot.

## 1. What this is

`gonetsurf` is a pure-Go web browser engine. The module path is `gonetsurf` (no host
prefix) and the target Go version is 1.27. Runtime dependencies are limited to the
standard library and `golang.org/x/*`; in-tree forks are permitted and must record their
upstream revision.

Work is organized into Phases 0–10 (see [`implementation-plan.md`](./implementation-plan.md))
and tracked task-by-task in [`task-tracking.md`](./task-tracking.md).

## 2. Current status

| Phase | Scope | State |
|---|---|---|
| 0 | Foundations (module, deps policy, panic barrier, fuzzing, docs) | **Done** (2 Verified, 3 Implemented) |
| 1 | WHATWG URL, encoding sniffing, HTML microsyntaxes | **Done** (10/10 Verified) |
| 2 | HTTP/1.1, TLS, DNS, cache, cookies, SOCKS, WebSocket, internal schemes | **Done** (18 Verified, 1 Implemented/Partial) |
| 3 | HTML parsing (`x/net/html`) + `dom` | **Done** (6/6 Verified) |
| 4 | CSS tokenizer → parser → selectors → values → CSSOM → cascade | **Done** (14/14 Verified) |
| 5 | Style resolution, UA sheet, selector matching, computed values | **Done** (6/6 Verified) |
| 6 | Layout: box tree, box model, block/inline/table/flex/grid, positioning, text | **In progress** (1 Verified, 8 Implemented/Partial, 5 Planned) |
| 7–10 | Paint/window/UI/process sandbox/JS/Web APIs | **Not started** |
| X | Cross-cutting backlog (forms, downloads, print, a11y, devtools) | **Not started** |

Totals: **57 Verified, 12 Implemented/Partial, 45 Planned, 0 Blocked**. There are no open
bugs recorded.

There is **no browser executable yet**. Phase 0–3 deliverables are the engine's
foundational libraries; the first runnable windowed browser is expected after Phases 4–8.

## 3. Quick start

```sh
go build ./...                          # compile everything
go vet ./...                            # static checks
gofmt -l cmd urls parsers protocols caches storages internal dom styles layouts   # formatting
go test ./... -count=1                  # unit + integration tests
go run ./cmd/netsurfctl deps-check      # enforce the no-third-party policy
go run ./cmd/netsurfctl urlparse 'https://www.example.com:8443/a/b?x=1#f'
go test ./urls/ -run TestURLParserConformance -v
go test ./parsers/css/tokenizer/ -run TestTokenizerConformance -v
go test ./urls/ -run=^$ -fuzz=FuzzParse -fuzztime=30s
```

Current verification baseline:

- 176 Go files across 37 packages, 65 test files, 299 test/fuzz/benchmark functions.
- `go test ./...` green for all packages; `go vet ./...` clean; `gofmt` clean.
- WHATWG URL conformance: **896/896** `urltestdata.json` cases pass.
- CSS tokenizer conformance: **223/223** reference tokenization cases pass.
- CSS parser conformance: all `css-parsing-tests` entry points pass. Declaration-value
  whitespace is normalized to the current syntax; two non-ASCII identifier cases are
  documented skips (`parsers/css/parser/testdata/SKIPS.md`).
- Selectors: **270** WPT validity cases pass with 78 tentative-feature skips.
- Property registry: **353** properties are generated from the pinned specification
  propdef tables, WPT grammar assertions, and explicit overrides (`cmd/propertygen`).
- Property values: the stable Phase 4 WPT set (**2,780** cases) reports **2,520 passes,
  0 failures, 260 documented skips**. Skips are listed in
  `parsers/css/properties/testdata/SKIPS.md` (Color 5, experimental sizing/overflow,
  tentative grid-lanes, and deferred deep shorthand/image-function validation).
- Values/math: a 29-case CSS Values 4 math corpus passes, along with unit-conversion
  tests for absolute lengths, angles, and times.
- Fuzzing: new targets cover the grammar engine, selectors, values/math, CSSOM, and
  property validation; short runs completed >190k executions each without a crash.
- Benchmarks cover tokenization, parsing, selectors, colors, grammar matching, values,
  CSSOM serialization, property validation, and the cascade.
- Conditions: `@supports`/`@import` corpus passes, including nested `not`, selector
  validation, layers, and media preludes.
- Media queries: environment matrix covers range syntax, boolean features, color,
  monochrome, grid, pointer/hover, preferences, display mode, dynamic range, scripting,
  gamut, and viewport segments.
- Animations: WPT-derived shorthand corpus passes; keyframes, lists, timing, and
  conflict validation are tested.
- Nesting: Go-native flattening corpus passes for `&`, implicit nesting, selector lists,
  and nested declarations.
- CSSOM: `CSSStyleSheet`, `CSSStyleDeclaration`, `MediaList`, rule insertion/deletion,
  nested rules, selector-text mutation, and serialization pass their tests.
- Cascade/variables: custom-property inheritance, `var()` fallback/cycles, `all`,
  `revert`, and `revert-layer` pass the Go-native harness ported from WPT cascade and
  variables cases.
- Selector matching: the full Selectors Level 4 matcher is wired into `dom.Matches`,
  `QuerySelector`, and `QuerySelectorAll`; combinators, attributes, structural and
  logical pseudo-classes, `:has()`, `:scope`, languages, and states are covered.
- Style resolution: `styles.Matcher` maintains id/class/tag/attribute indexes and
  document order; the style engine applies the embedded UA sheet, presentational
  hints, author/inline origins, nesting, `@media`/`@supports`, inheritance, `all`,
  and custom properties/`var()`.
- Computed values: font-size keywords/`em`/`rem`/viewport units, font-weight
  bolder/lighter, line-height percentages, color/`currentcolor`, opacity, border
  widths, and generic length/angle/time conversion are tested.
- Incremental restyle: DOM mutation versions and dirty-node tracking drive cached
  subtree restyles; `:has()` stylesheets fall back to a full restyle.
- Layout: box-tree construction, anonymous block boxes, `display:contents`, box
  model/sizing (including intrinsic min-/max-content and fit-content), block
  layout with sibling and parent/child margin collapsing, simple inline line
  layout with word wrapping, relative/absolute/fixed positioning, and basic
  row/column flexbox with grow/shrink, alignment, gaps, and order are tested.
  Basic table layout covers intrinsic column sizing, width distribution, row
  heights, and cell layout. Basic grid layout covers explicit/`repeat()` tracks,
  `fr` distribution, gaps, auto-placement, and explicit line placement. Floats
  cover left/right placement, shrink-to-fit widths, and block avoidance.
  Shorthand expansion covers margin, padding, border, border components, overflow,
  flex, flex-flow, and gap.
- Matching benchmarks: `dom` selector matching and `styles.Matcher.Elements` are
  benchmarked; the DOM matcher also has a fuzz target.
- An+B conformance: **128/128** cases; `<urange>`: **12/12** cases.
- Color 4: hex, keyword, HSL, HWB, lab/lch/oklab/oklch, and `color()`
  corpora pass (**~7,300** cases).
- IDNA: **2669** cases pass; encoding sniffing: **81** cases pass.
- HTML tree construction: **1578** cases pass (192 fragment contexts plus
  documented parser divergences skipped).
- HTTP/1.1: framing/smuggling corpus and a `net/http` differential test pass;
  `FuzzReadResponse` ran 1.7M executions without a crash.
- DNS-over-TLS: local TLS DNS server test passes. WebSocket: opening handshake,
  fragmentation, control frames, close codes, masking, and UTF-8 validation pass.
- `netsurfctl fetch <url>` retrieves a URL end to end.
- Verification is run through `make compliance-tests`.
- Fuzzing: `FuzzCSS` ran 375k executions without a crash.
- `deps-check` confirms only `golang.org/x/net` and `golang.org/x/text` (plus stdlib) are
  in the dependency graph.

## 4. Repository map (implemented)

| Path | What it does |
|---|---|
| `urls/` | WHATWG URL parser/serializer, origin, site, IDNA, internal schemes, `data:` decoding. |
| `parsers/html/` | `x/net/html` wrapper: encoding → parse → `dom` conversion, node budget, quirks detection, fragment parsing/serialization. |
| `parsers/html/encodings/` | BOM/transport/meta-prescan encoding sniffing and re-decode. |
| `parsers/html/microsyntaxes/` | HTML numeric, date/time, color, token-list, and reference parsers. |
| `parsers/css/tokenizer/` | CSS Syntax 3 tokenizer: preprocessing, all token kinds/flags, per-token `Representation`. |
| `parsers/css/parser/` | Component values, functions, simple blocks, rules, declarations, entry points. |
| `parsers/css/grammar/` | Value-definition grammar engine (`\|`, `\|\|`, `&&`, multipliers, ranges). |
| `parsers/css/selectors/` | Selectors Level 4 parsing, specificity, and the An+B microsyntax. |
| `parsers/css/values/` | Numeric units, unit conversion, `calc()`/`min()`/`max()`/`clamp()`, `<urange>`. |
| `parsers/css/colors/` | Color 4 parsing/serialization: hex, named, rgb/hsl/hwb, lab/oklch/color(). |
| `parsers/css/cssom/` | Stylesheet, style rule, at-rule, and declaration objects with serialization. |
| `parsers/css/cascade/` | Origin/importance/layer/specificity/order cascade and inheritance. |
| `parsers/css/conditions/` | `@supports` conditions and `@import` descriptors. |
| `parsers/css/mediaqueries/` | Media Queries 5 parsing and environment evaluation. |
| `parsers/css/properties/` | Property registry: inheritance, initial, grammar, animation metadata. |
| `parsers/css/animations/` | `@keyframes` and the animation shorthand. |
| `parsers/css/nesting/` | CSS Nesting parsing and `&` resolution. |
| `dom/` | Namespace-aware mutable DOM: nodes, documents, attributes, queries, templates, shadow scaffolding, sandbox flags. |
| `styles/` | Indexed selector matcher, UA stylesheet, presentational hints, and the style-resolution engine. |
| `layouts/` | Box-tree construction, box model, block layout, and simple inline line layout. |
| `protocols/http1/` | Custom HTTP/1.1 client: ordered headers, strict framing, chunked+trailers, pool, redirects, conditional/range. |
| `protocols/tls/` | TLS 1.3 client config, ALPN, ECH field, GREASE values. |
| `protocols/dns/` | DNS codec (compression), caching resolver, special-use names, DoH, DoT. |
| `protocols/socks/` | SOCKS5 CONNECT client. |
| `protocols/websockets/` | RFC 6455 accept key and masked frame codec. |
| `protocols/mime/` | Content-Disposition parsing and filename sanitization. |
| `protocols/internal/` | `about:`, `data:`, `netsurf-cache:` handlers. |
| `caches/disk/` | Invertible URL↔filesystem mapping and JSONL journal/redirect chain. |
| `caches/http/` | RFC 7234 policy plus the aggressive archive mode. |
| `caches/memory/` | Generic in-memory LRU. |
| `storages/cookies/` | RFC 6265 partitioned jar, cross-site dependency graph, CNAME-cloaking tracking-session reset. |
| `internal/safecall/` | Panic-to-error boundary helpers. |
| `cmd/netsurfctl/` | `deps-check` and `urlparse` maintenance commands. |
| `cmd/corpusgen/` | Refresh and normalize the vendored WPT, URL, HTML, and CSS corpora. |
| `cmd/propertygen/` | Generate the property registry from pinned specs and WPT grammar assertions. |

The target layout (including not-yet-created packages such as `rendering/`, `windowing/`,
`webapis/`, `runtime/`, `brokers/`, `engines/`, `ipc/`, `farbling/`, and
`javascripts/sobek/`) is described in
[`architecture.md`](./architecture.md). Those are design targets; only the packages in the
table above exist today.

## 5. Known limitations and partial work

These are intentional, tracked gaps rather than surprises:

- **Phase 4 is complete but deliberately scoped.** All Phase 4 tasks are `Verified`.
  Remaining CSS gaps are feature-depth gaps rather than task gaps: the CSSOM covers the
  core object model but not the full IDL (constructable/adopted stylesheets, every rule
  subclass), media queries do not implement serialization or every experimental feature,
  and cascade `revert-layer` follows the common origin/layer cases rather than every
  edge.
- **Computed values cover the style engine's supported property set.** Font, color,
  line-height, opacity, border widths, and generic length/angle/time conversion are
  implemented. Layout-specific used values (percentage resolution against containing
  blocks, intrinsic sizing, text metrics) belong to Phase 6.
- **Layout is at its foundation stage.** P6-LAYOUT-002/003/004/005/006/007/008/009 cover
  the box model, intrinsic sizing, sibling and parent/child margin collapsing, simple line
  wrapping, basic tables, floats, relative/absolute/fixed positioning, basic flexbox, and
  basic grid. Flex wrapping and baseline alignment, sticky positioning, writing modes,
  bidi, text shaping, and overflow/scroll containers remain planned Phase 6 tasks.
- **Incremental restyle is conservative.** Dirty nodes restyle their parent subtree;
  stylesheets using `:has()` trigger a full restyle because descendant changes can
  invalidate ancestors. Selector indexes are rebuilt on restyle.
- **CSS parsing conformance corpus.** The public `css-parsing-tests` corpora target a
  pre-2021 tokenizer (match tokens such as `~=`, a `unicode-range` token, and normalized
  number representations). `cmd/corpusgen` normalizes the expected values for the pinned
  syntax; two non-ASCII identifier cases remain documented skips. The tokenizer, An+B,
  `<urange>`, and Color 4 corpora are vendored and pass.
- **Property registry and value skips.** The registry is generated for 353 properties;
  the stable value corpus excludes experimental/tentative properties and records 260
  skips (Color 5, experimental sizing/overflow, tentative grid-lanes, and deferred deep
  shorthand/image-function validation). The generator warns and skips four malformed
  spec grammar tables (`columns`, `inset-block-start`, `margin-block-start`,
  `padding-block-start`).
- **Color 4 lab/lch/oklab/oklch/color()** parse and serialize in their declared space, but
  gamut mapping and full reference serialization are not implemented or conformance-tested.
- **CSS identifier definitions.** The pinned `specifications/css/syntax-3.html` is the 2021
  Candidate Recommendation, which predates the current "non-ASCII ident code point" ranges.
  The tokenizer follows the current CSS Syntax 3 definition (matching WPT and browsers); the
  pinned spec should be refreshed with `specifications/update.sh`.
- **P2-TLS-002 (ECH/GREASE)** — GREASE values and the ECH config field are present, but
  `crypto/tls` cannot inject GREASE or permute the ClientHello. That needs the planned
  `protocols/tls/chrome` fork.
- **P1-MICRO-003 (colors)** — the legacy color parser recognizes the basic HTML color set;
  the full CSS named-color table is deferred to the CSS color parser.
- **`x/net/html` namespaces** — the upstream parser uses short namespace names (`""`,
  `svg`, `math`); `parsers/html` maps these to the DOM namespace URLs and back.
- **No `sobek` fork yet.** The JavaScript engine is a Phase 10 deliverable. When vendored,
  its non-essential third-party dependencies (`sourcemap`, `pprof`, `yaml`, and possibly
  `semver`) must be pruned and `regexp2` replaced per the project decision.
- **No in-tree forks exist yet**, so no `FORK.md` files are present.
- **Specifications tree is ~50 MB** and committed for offline verification. See
  [`specifications.md`](./specifications.md) §5 for the LFS option.

## 6. Decisions that are already locked

Recorded in the relevant design docs under "Decisions":

- Module path `gonetsurf`; Go 1.27; stdlib + `golang.org/x/*` only.
- DOM package is `dom` (singular); category packages are plural.
- The WHATWG URL parser in `urls` is the only URL parser used for security/cache keys.
- No single-letter identifiers; descriptive names; comments only where they add value.
- Aggressive archive caching with a user-visible archive banner
  ([`cache.md`](./cache.md)).
- Persistent, partitioned cookie jar with a cross-site dependency graph and atomic
  tracking-session reset that defeats CNAME-cloaked trackers
  ([`privacy.md`](./privacy.md)).
- OS process per site/tab with a trusted network broker, sandboxed with
  namespaces/seccomp/Landlock/rlimits/cgroups ([`security.md`](./security.md)).
- Farbling is deterministic per origin+session via an HMAC of an install seed.

## 7. Where to look

| Question | Document |
|---|---|
| What are we building and in what order? | [`implementation-plan.md`](./implementation-plan.md) |
| What is done / tested / blocked? | [`task-tracking.md`](./task-tracking.md) |
| How is the engine structured? | [`architecture.md`](./architecture.md) |
| How does the cache map URLs to disk? | [`cache.md`](./cache.md) |
| How is privacy/tracking handled? | [`privacy.md`](./privacy.md) |
| How is untrusted content isolated? | [`security.md`](./security.md) |
| Coding and test conventions | [`conventions.md`](./conventions.md) |
| Which specs are pinned and how to refresh | [`specifications.md`](./specifications.md) |

## 8. Recommended next steps

1. **Continue Phase 6 — Layout.** Add sticky positioning, flex wrapping/baseline
   alignment, writing modes, bidi, text metrics, overflow/scroll, and incremental
   layout.
2. **Deepen CSS feature coverage** as Phase 6/7 need it: full CSSOM IDL surface, media
   query serialization and remaining features, and cascade edge cases.
3. **Close the remaining Partial network task** (TLS GREASE/ECH) when the
   `protocols/tls/chrome` fork lands.
4. **Fill the skeleton.** Create the remaining package directories with `doc.go` stubs as
   their phases begin; avoid broad empty scaffolding ahead of need.

## 9. Handover checklist

- [x] Builds, vets, formats, and tests clean.
- [x] Task ledger reflects reality and has no stale "no code yet" claims.
- [x] Design docs describe both the target architecture and the current state.
- [x] Known limitations are listed with their task IDs.
- [x] No secrets, credentials, or generated artifacts are committed.
- [ ] Everything under `docs/` reviewed by the next maintainer.

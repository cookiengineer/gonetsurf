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

> **Current state (2026-10-04):** Phases 0–4 are implemented and their
> verification is well advanced. 146 Go files across 34 packages, 52 test files
> with 205 test/fuzz/benchmark functions; `make compliance-tests` green. The CSS
> specifications were refreshed to the current editor's drafts, and external
> conformance now covers the URL corpus (896/896), CSS tokenizer (223/223),
> parser, selectors, values/properties, Color 4 (~7,300), IDNA (2669), encoding
> sniffing (81), and HTML tree construction (1578), plus HTTP framing/differential,
> DoT, and WebSocket handshake tests. Phase 5 (style resolution) is next.
> For a narrative summary see [`handover.md`](./handover.md).

## Summary

| Phase | Tasks | Verified | Implemented/WIP | Planned | Blocked |
|---|---:|---:|---:|---:|---:|
| 0 Foundations | 5 | 2 | 3 | 0 | 0 |
| 1 URL/Encoding/Microsyntax | 10 | 10 | 0 | 0 | 0 |
| 2 Network/Cookies/Cache | 19 | 18 | 1 | 0 | 0 |
| 3 HTML/DOM | 6 | 6 | 0 | 0 | 0 |
| 4 CSS/CSSOM | 14 | 2 | 12 | 0 | 0 |
| 5 Style | 6 | 0 | 0 | 6 | 0 |
| 6 Layout | 14 | 0 | 0 | 14 | 0 |
| 7 Paint/Fonts/Images/Color | 10 | 0 | 0 | 10 | 0 |
| 8 Window/UI | 6 | 0 | 0 | 6 | 0 |
| 9 Process/Sandbox | 6 | 0 | 0 | 6 | 0 |
| 10 JS/Web APIs/Platforms | 10 | 0 | 0 | 10 | 0 |
| Cross-cutting | 8 | 0 | 0 | 8 | 0 |
| **Total** | **114** | **38** | **16** | **60** | **0** |

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
| P4-CSS-002 | Parser and component values | Implemented | Y | P | Partial | - |
| P4-CSS-003 | Value-definition grammar engine | Implemented | Y | P | Partial | - |
| P4-CSS-004 | Selectors Level 4 | Implemented | Y | P | Partial | - |
| P4-CSS-005 | Values, units, and math | Implemented | Y | P | Partial | - |
| P4-CSS-006 | Color 4 | Verified | Y | Y | Compliant | - |
| P4-CSS-007 | CSSOM | Implemented | Y | P | Partial | - |
| P4-CSS-008 | Cascade and inheritance | Implemented | Y | P | Partial | - |
| P4-CSS-009 | Conditional rules | Implemented | Y | P | Partial | - |
| P4-CSS-010 | Media queries | Implemented | Y | P | Partial | - |
| P4-CSS-011 | Property registry | Implemented | Y | P | Partial | - |
| P4-CSS-012 | Animations | Implemented | Y | P | Partial | - |
| P4-CSS-013 | CSS Nesting | Implemented | Y | P | Partial | - |
| P4-CSS-014 | Fuzzing and performance | Implemented | Y | N/A | N/A | - |

## Phase 5 — Style Resolution

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P5-STYLE-001 | UA stylesheet and presentational hints | Planned | - | - | Unassessed | - |
| P5-STYLE-002 | Selector matching engine | Planned | - | - | Unassessed | - |
| P5-STYLE-003 | Computed value resolution | Planned | - | - | Unassessed | - |
| P5-STYLE-004 | Custom properties and var() substitution | Planned | - | - | Unassessed | - |
| P5-STYLE-005 | Media/conditional evaluation integration | Planned | - | - | Unassessed | - |
| P5-STYLE-006 | Incremental restyle | Planned | - | - | N/A | - |

## Phase 6 — Layout

| ID | Task | Status | UT | E2E | Spec | Bugs |
|---|---|---|---|---|---|---|
| P6-LAYOUT-001 | Box tree construction | Planned | - | - | Unassessed | - |
| P6-LAYOUT-002 | Box model and intrinsic sizing | Planned | - | - | Unassessed | - |
| P6-LAYOUT-003 | Block formatting context | Planned | - | - | Unassessed | - |
| P6-LAYOUT-004 | Inline formatting context | Planned | - | - | Unassessed | - |
| P6-LAYOUT-005 | Table formatting context | Planned | - | - | Unassessed | - |
| P6-LAYOUT-006 | Floats | Planned | - | - | Unassessed | - |
| P6-LAYOUT-007 | Positioning | Planned | - | - | Unassessed | - |
| P6-LAYOUT-008 | Flexbox | Planned | - | - | Unassessed | - |
| P6-LAYOUT-009 | Grid | Planned | - | - | Unassessed | - |
| P6-LAYOUT-010 | Writing modes and logical properties | Planned | - | - | Unassessed | - |
| P6-LAYOUT-011 | Bidi and text runs | Planned | - | - | Unassessed | - |
| P6-LAYOUT-012 | Text metrics and simple shaping | Planned | - | - | Unassessed | - |
| P6-LAYOUT-013 | Overflow and scrolling | Planned | - | - | Unassessed | - |
| P6-LAYOUT-014 | Incremental layout | Planned | - | - | N/A | - |

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

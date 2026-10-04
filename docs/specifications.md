# Specification Inventory and Provenance

The `specifications/` directory holds pinned snapshots of the normative documents the
engine is implemented against. This document records what is present, where it came from,
and what is still missing.

## 1. Rules

1. Specifications are **pinned snapshots**. Do not edit them by hand.
2. Fetch only from canonical publishers: WHATWG, W3C/CSSWG, IETF.
3. Refresh with `sh specifications/update.sh`, then update the "Fetched" dates in this
   document.
4. The refresh script only downloads files that are absent; delete a file to re-fetch it.
5. Implementation tasks cite the specific file and section, e.g.
   `specifications/css/syntax-3.html` §4.3.1.

## 2. Inventory

Fetched on **2026-10-04**. The CSS specifications were refreshed to the current
editor's drafts on the same day (see "Refreshes" below). `specifications/update.sh`
now covers every single-file CSS specification; pass `REFRESH=1` to re-fetch
files that already exist.

### Refreshes

| Date | Documents | Material changes |
|---|---|---|
| 2026-10-04 | All `css/*.html` | CSS Syntax 3 re-added `<unicode-range-token>` (§4.3.11, §4.3.14) and the `unicode ranges allowed` parameter; "non-ASCII ident code point" ranges are now restricted; parser entry points reorganized (§5.4/§5.5). `update.sh` now fetches `syntax-3`, `cascade-4`, `conditional-rules-3`, `media-queries-5`, `values-and-units-4`, `color-4`, `animations-1`, `backgrounds-and-borders-3`, `images-4`, and `paged-media-3`. |

### CSS

| File | Document | Source |
|---|---|---|
| `css/syntax-3.html` | CSS Syntax Level 3 | CSSWG draft |
| `css/cascade-4.html` | CSS Cascading and Inheritance Level 4 | CSSWG draft |
| `css/cascade-5.html` | CSS Cascading and Inheritance Level 5 | https://drafts.csswg.org/css-cascade-5/ |
| `css/conditional-rules-3.html` | CSS Conditional Rules Level 3 | CSSWG draft |
| `css/media-queries-5.html` | Media Queries Level 5 | CSSWG draft |
| `css/values-and-units-4.html` | CSS Values and Units Level 4 | CSSWG draft |
| `css/color-4.html` | CSS Color Level 4 | CSSWG draft |
| `css/selectors-4.html` | Selectors Level 4 | https://drafts.csswg.org/selectors-4/ |
| `css/css-display-3.html` | CSS Display Level 3 | https://drafts.csswg.org/css-display-3/ |
| `css/css-box-3.html` | CSS Box Model Level 3 | https://drafts.csswg.org/css-box-3/ |
| `css/css-sizing-3.html` | CSS Box Sizing Level 3 | https://drafts.csswg.org/css-sizing-3/ |
| `css/css-position-3.html` | CSS Positioned Layout Level 3 | https://drafts.csswg.org/css-position-3/ |
| `css/css-overflow-3.html` | CSS Overflow Level 3 | https://drafts.csswg.org/css-overflow-3/ |
| `css/css-flexbox-1.html` | CSS Flexible Box Layout Level 1 | https://drafts.csswg.org/css-flexbox-1/ |
| `css/css-grid-2.html` | CSS Grid Layout Level 2 | https://drafts.csswg.org/css-grid-2/ |
| `css/css-writing-modes-4.html` | CSS Writing Modes Level 4 | https://drafts.csswg.org/css-writing-modes-4/ |
| `css/css-text-3.html` | CSS Text Level 3 | https://drafts.csswg.org/css-text-3/ |
| `css/css-fonts-4.html` | CSS Fonts Level 4 | https://drafts.csswg.org/css-fonts-4/ |
| `css/css-variables-1.html` | CSS Custom Properties Level 1 | https://drafts.csswg.org/css-variables-1/ |
| `css/css-nesting-1.html` | CSS Nesting Level 1 | https://drafts.csswg.org/css-nesting-1/ |
| `css/cssom-1.html` | CSS Object Model | https://drafts.csswg.org/cssom/ |
| `css/cssom-view-1.html` | CSSOM View | https://drafts.csswg.org/cssom-view/ |
| `css/css-transforms-1.html` | CSS Transforms Level 1 | https://drafts.csswg.org/css-transforms-1/ |
| `css/css2-2/` | CSS Level 2 Revision 2 (multi-page) | https://www.w3.org/TR/CSS22/ |
| `css/animations-1.html` | CSS Animations Level 1 | CSSWG draft |
| `css/backgrounds-and-borders-3.html` | CSS Backgrounds and Borders Level 3 | CSSWG draft |
| `css/images-4.html` | CSS Image Values and Replaced Content Level 4 | CSSWG draft |
| `css/paged-media-3.html` | CSS Paged Media Level 3 | CSSWG draft |

### Web platform

| File | Document | Source |
|---|---|---|
| `web/url.html` | URL Standard | https://url.spec.whatwg.org/ |
| `web/encoding.html` | Encoding Standard | https://encoding.spec.whatwg.org/ |
| `web/mimesniff.html` | MIME Sniffing Standard | https://mimesniff.spec.whatwg.org/ |
| `web/infra.html` | Infra Standard | https://infra.spec.whatwg.org/ |
| `web/dom.html` | DOM Standard | https://dom.spec.whatwg.org/ |
| `web/fetch.html` | Fetch Standard | https://fetch.spec.whatwg.org/ |
| `web/referrer-policy.html` | Referrer Policy | https://w3c.github.io/webappsec-referrer-policy/ |
| `web/mixed-content.html` | Mixed Content | https://w3c.github.io/webappsec-mixed-content/ |
| `web/csp-3.html` | Content Security Policy Level 3 | https://w3c.github.io/webappsec-csp/ |
| `web/service-workers.html` | Service Workers | https://w3c.github.io/ServiceWorker/ |
| `web/svg2-single-page.html` | SVG 2 (single page) | https://www.w3.org/TR/SVG2/single-page.html |
| `web/ecmascript.html` | ECMAScript Language Specification | https://tc39.es/ecma262/ |

### HTML and legacy HTML

Present under `html/`: WHATWG HTML split sections (parsing, infrastructure, DOM,
origin, rendering, forms, tables, semantics, syntax, named characters, embedded content,
and others).

### Networking

| File | Document |
|---|---|
| `http/rfc7230.txt` … `rfc7234.txt` | HTTP/1.1 core, semantics, conditional, range, caching. |
| `http/rfc6265.txt` | HTTP State Management Mechanism (cookies). |
| `http/rfc5861.txt` | HTTP Cache-Control Extensions for Stale Content. |
| `http/rfc2109.txt` | HTTP State Management (legacy). |
| `http/rfc6266.txt` | Content-Disposition. |
| `http/rfc2295.txt` | Transparent Content Negotiation. |
| `http/rfc6585.txt` | Additional HTTP Status Codes. |
| `http/rfc2616.txt` | HTTP/1.1 (historic). |
| `dns/rfc1034.txt`, `rfc1035.txt`, `rfc2181.txt`, `rfc6891.txt`, `rfc7766.txt`, `rfc3596.txt`, `rfc6761.txt`, `rfc8484.txt`, `rfc7858.txt` | DNS core, EDNS(0), TCP, AAAA, special-use names, DoH, DoT. |
| `dns/rfc4033.txt`, `rfc4034.txt`, `rfc4035.txt` | DNSSEC core. |
| `mdns/rfc6762.txt`, `rfc6763.txt` | Multicast DNS and DNS-SD. |
| `socks/rfc1928.txt` | SOCKS Protocol Version 5. |
| `ws/rfc6455.txt` | WebSocket Protocol. |
| `other/rfc8446.txt` | TLS 1.3. |
| `other/rfc8701.txt` | GREASE. |
| `other/rfc3339.txt` | Date and Time on the Internet. |
| `other/rfc4648.txt` | Base16/32/64. |
| `other/rfc7830.txt` | EDNS(0) Padding Option. |
| `other/rfc7673.txt` | mDNS ICE candidates. |
| `other/rfc9113.txt` | HTTP/2. |
| `other/rfc7541.txt` | HPACK. |
| `other/rfc7301.txt` | TLS ALPN. |
| `other/rfc6066.txt` | TLS extensions (SNI). |
| `other/rfc6797.txt` | HSTS. |
| `other/rfc7692.txt` | WebSocket Per-Message Deflate. |
| `other/multicast-addresses.txt` | IANA multicast addresses. |

## 3. Still missing / deferred

These were not fetched because they are not single-file documents, are not yet needed, or
require manual acquisition. Add them before the phase that needs them.

| Needed for | Document | Note |
|---|---|---|
| P8-X11 | X11 protocol / XCB | Not a single canonical file; implement from the protocol reference. |
| P10-WAYLAND | Wayland protocol | XML protocol files; vendor under `specifications/wayland/` when needed. |
| P10-TEXT-001 | OpenType / GSUB / GPOS | Microsoft multi-page reference; acquire or summarize manually. |
| X-ACC-001 | HTML Accessibility API Mappings, ARIA | W3C documents; acquire before accessibility phase. |
| P7-COLOR-001 | ICC / color management | Not in stdlib; may remain out of scope. |
| P10-NET-001 | QUIC / HTTP/3 (`rfc9000`, `rfc9114`) | Only if HTTP/3 is pursued. |
| P10-JS-001 | Test262 conformance suite | Test data, not a spec; vendor separately with test harness. |
| P1-URL-001 | WHATWG URL test data (`urltestdata.json`) | Test data for the URL parser. |
| P4-CSS-* | WPT / CSSWG test suites (`csswg-test`, `wpt`) | Test data; vendor with each phase's conformance harness. |
| P3-HTML-001 | html5lib-tests | Test data; `x/net/html` already ships a suite. |

## 4. Refresh procedure

```sh
sh specifications/update.sh
```

Then update the "Fetched on" date above and note any document that changed materially.
Because the downloaded files are pinned, a refresh requires deleting the specific file
first.

## 5. Size note

The specification tree is approximately 50 MB. It is intentionally committed so the
implementation can be built and verified offline. If repository size becomes a concern,
move the tree to a Git LFS or a separately checked-out `specifications/` volume and record
the change in this document.

## 6. Decisions

- **D-SPEC-001:** Only canonical publishers are used; no third-party mirrors.
- **D-SPEC-002:** Multi-page documents (CSS 2.2, SVG 2) are stored as complete page sets or
  their single-page variant.
- **D-SPEC-003:** `specifications/update.sh` is the only supported refresh path.

## 7. Vendored test corpora

Offline conformance data lives under `testdata/upstream/` and is committed so
that `go test` never requires the network. `cmd/corpusgen` reads it and writes
the normalized corpora consumed by the package tests (`make corpus-refresh`
re-downloads the raw sources first).

| Directory | Upstream | Used by |
|---|---|---|
| `testdata/upstream/css-parsing-tests/` | CourtBouillon/css-parsing-tests | CSS tokenizer, parser, selectors (An+B), colors |
| `testdata/upstream/wpt/url/resources/` | web-platform-tests/wpt | URL and IDNA conformance |
| `testdata/upstream/wpt/acid/` | web-platform-tests/wpt | Acid2/Acid3 (Phase 7+) |
| `testdata/upstream/html5lib-tests/` | html5lib/html5lib-tests | HTML tokenizer, tree construction, encoding |

Acid1 is not vendored because its canonical location is currently unavailable;
see `testdata/upstream/acid/SOURCE`.

- **D-SPEC-004:** Vendored corpora are pinned with a `SOURCE` file recording the
  upstream URL, revision where known, license, and fetch date.

# Cache Design

The cache is both the performance layer and a **user-owned, exportable archive**. It is
optimized for durability and human readability, not for obedience to server-declared short
lifetimes.

## 1. Goals

1. **Aggressive by default.** A resource that has been fetched is kept and reused even if
   the server declared `no-cache`, `must-revalidate`, or a short `max-age`.
2. **Never silently stale.** When an archived resource is served past its server-declared
   freshness, the browser shows a persistent archive banner with its capture timestamp and
   a Reload action.
3. **Human-readable mirror.** A URL maps to a path the user recognizes, e.g.
   `https://example.com/what/ever/index.html` →
   `~/NetSurf/cache/example.com/what/ever/index.html`.
4. **Collision-safe.** Query strings and file/directory conflicts never destroy data.
5. **Exportable and reconstructible.** Copying the mirror alone is enough to browse it;
   the journal adds exact cache semantics.
6. **Partitioned.** The on-disk layout is partitioned by top-level site by default so the
   archive cannot be used for cross-site correlation.

## 2. Aggressive archive policy

### 2.1 Freshness override

For each response the cache computes normal RFC 7234 freshness, then applies the archive
policy:

| Server directive | Default archive behavior |
|---|---|
| `max-age`, `s-maxage`, `Expires` | Honor while fresh; retain after expiry. |
| `no-cache` | Store; serve from archive and mark stale; revalidate only on Reload. |
| `must-revalidate` / `proxy-revalidate` | Store; do not force revalidation on ordinary navigation. |
| `no-store` / `private` | Store in the **user archive** (local, single-user) but never share; still show the banner. |
| `Vary` | Keep per-variant entries; selection is metadata-driven. |
| `Set-Cookie` | Does not prevent archiving; cookies live in the partitioned jar. |

Authorization-bearing responses are archived locally but are never treated as reusable for
a different partition or a different user profile.

### 2.2 Archive metadata

Every archived resource stores:

- capture time (wall clock, RFC 3339),
- server `Date` and `Age` (if present),
- original server freshness lifetime and the computed `current_age`,
- response status, headers, validators (`ETag`, `Last-Modified`),
- content type and final URL after redirects.

### 2.3 Archive banner

When a resource is served from the archive past its server freshness, the shell shows:

> 📦 This is an archived version from `YYYY-MM-DD HH:ii:SS`. Press **Reload** to update it.

- The banner is per top-level document and persists until a successful refresh.
- **Reload** bypasses the archive, performs a network fetch with normal validators, and
  replaces the entry on success. On failure, the archive remains and the banner states
  that the update failed.
- Shift-Reload additionally bypasses all caches for the document and its subresources.
- The banner is never rendered by untrusted content and cannot be spoofed by a page.

### 2.4 Stale-while-revalidate

Optional background revalidation (RFC 5861) may refresh a stale entry while serving the
archive copy, but the banner stays until the refresh completes. Background revalidation is
disabled by default to avoid unsolicited network traffic.

## 3. On-disk layout

```
~/NetSurf/
├── cache/
│   ├── example.com/
│   │   ├── what/ever/index.html
│   │   └── index.php/
│   │       └── ~q~/
│   │           ├── category/index.html
│   │           └── category/subcategory/index.html
│   └── .netsurf/
│       ├── index.jsonl        # URL <-> path, headers, validators, partitions
│       ├── redirects.jsonl    # redirect chains and promotions
│       └── vary/              # variant metadata
├── cookies/
│   └── jar.jsonl              # partitioned cookie jar + tracking-session graph
├── storage/
├── config/
└── history/
```

The `.netsurf/` directory is hidden but travels with the tree in normal archives. It is
optional for browsing: the mirror alone is sufficient.

## 4. URL ↔ filesystem mapping

### 4.1 Algorithm

1. **Normalize** with `urls` (WHATWG). This is the only URL form used for mapping.
2. **Authority segment.**
   - `host` in IDNA ASCII.
   - Non-default port → `host~port`.
   - Scheme: omitted for `https` (the common case, matching the requested example);
     `http` → `~http~host`; any other scheme → `~scheme~host`.
   - `~` cannot occur in a DNS host, so authority escaping is unnecessary.
3. **Path segments.**
   - Use the normalized, percent-encoded path (reversible; readable for common paths).
   - Empty middle segment → `~empty~`.
   - A real segment that begins with the reserved sentinel `~` is escaped by doubling it
     (`~~`).
   - Segments longer than 240 bytes are truncated with a short hash suffix; the original
     is recorded in the journal.
4. **No query.**
   - Path ending in `/` (or empty) → append `index.html`.
   - Otherwise the last segment is a file. Its kind is recorded, and it is promoted to
     `<seg>/index.html` if a directory of the same name is later required.
5. **With query.**
   - Append the reserved directory `~q~`, then the encoded query as sub-segments.
   - `/` in the query → `%2F`; `&` and `=` are kept; `~` is doubled.
   - Over-long queries are split into `~q~/part0~part1/...`; the split rule is recorded.
6. **Fragment.** Never part of the path; stored in metadata for scroll restoration.

### 4.2 Worked examples

| URL | Local path |
|---|---|
| `https://example.com/what/ever/index.html` | `example.com/what/ever/index.html` |
| `https://whatever.com/index.php?category` | `whatever.com/index.php/~q~/category/index.html` |
| `https://whatever.com/index.php?category/subcategory` | `whatever.com/index.php/~q~/category/subcategory/index.html` |
| `https://example.com/a?b=c` | `example.com/a/~q~/b=c/index.html` |
| `https://example.com/?q=x` | `example.com/~q~/q=x/index.html` |
| `http://example.com/` | `~http~example.com/index.html` |
| `https://example.com:8443/x` | `example.com~8443/x` |

### 4.3 Collision and quirk handling

| Situation | Resolution |
|---|---|
| `a` is a file and `a/b` is needed | Promote `a` → `a/index.html`; record the promotion. |
| Two URLs differ only by case on a case-insensitive FS | Disambiguate one as `~case~<hash>`. |
| Reserved device names (`CON`, `NUL`, `COM1`, …) | Prefix segment with `~r~`. |
| Trailing space or dot in a segment | Escape with `~r~`. |
| Multiple query orderings of the same parameters | Distinct entries; the journal records the exact URL. |
| Same host over both `http` and `https` | Scheme sentinel separates them. |

## 5. Redirects, promotions, and quirks

- `.netsurf/redirects.jsonl` records `{from, to, status, at}` and promotion events.
- Resolving a cached URL follows the redirect chain to the final entry.
- When exporting to a machine **without** the journal, a redirect stub is written at the
  source path:
  ```html
  <!-- netsurf-cache-redirect -->
  <meta http-equiv="refresh" content="0; url=<target>">
  ```
  so the tree remains navigable.
- HSTS upgrades (`http`→`https`), trailing-slash canonicalization, and
  `Content-Location` are recorded the same way.

## 6. Variants (`Vary`)

- Variants live under `<resource>/~vary~/<vary-hash>/`.
- The vary hash is computed over the normalized selecting header values recorded in
  metadata.
- `Vary: *` makes the resource non-reusable but still archived in its own entry.

## 7. Concurrency and durability

- Writes are atomic: write to a temporary file in the target directory, then `rename`.
- Optional `fsync` for durability; configurable for performance.
- A per-origin lock serializes writers in the broker.
- Journal appends are crash-safe (line-oriented, truncated on partial writes).
- Single-flight deduplication prevents duplicate concurrent writes for the same URL.

## 8. Export, import, and verification

`netsurfctl` provides:

- `cache export <dir>` — copy the mirror and journal with consistent metadata.
- `cache import <dir>` — rebuild the journal by scanning the mirror; verify invertibility;
  report collisions and unreadable entries.
- `cache verify` — property-check `PathToURL(URLToPath(u)) == u` over the journal and the
  filesystem; detect orphans and dangling redirects.
- `cache clear --origin <origin>` and `cache clear --site <site>` — partition-aware
  deletion.

## 9. Partitioning

- Default partition key: registrable domain of the top-level site (`x/net/publicsuffix`).
- The key is stored in metadata, not in the mirrored path, so the export stays readable.
- A strict per-origin mode writes partitions as `~site~<registrable-domain>/...`.
  The exact placement is a Phase 2 decision recorded in the journal.

## 10. Privacy interactions

- The archive is local and single-user; it is not a shared proxy cache.
- Cross-site resources are stored under the first-party partition that fetched them, so a
  third party cannot enumerate its own footprint across sites.
- Clearing a tracking session also clears the associated archived entries on request (see
  `privacy.md`), while preserving the user's explicit long-term archive.

## 11. Decisions

- **D-CACHE-001:** Ignore short server lifetimes by default; serve the archive with a
  visible banner.
- **D-CACHE-002:** `~/NetSurf/cache/` is the human-readable mirror; `.netsurf/` inside it
  is the journal.
- **D-CACHE-003:** Query strings map under the reserved `~q~` directory, not into the path,
  to avoid file/directory collisions.
- **D-CACHE-004:** Redirects are journaled and stubbed so exports remain browsable.
- **D-CACHE-005:** Partition metadata is kept out of the visible path for readability.

# Privacy, Tracking Sessions, and Farbling

Privacy in `gonetsurf` is enforced across the transport, storage, and script layers. This
document defines the tracking-session cookie model, the CNAME-cloaking defense, the
network privacy defaults, and the Web API farbling contract.

## 1. Principles

1. **The server is not the wiser.** After a tracking-session reset (e.g. Reload), a site
   must be unable to correlate the new session with the previous one via cookies,
   storage, or cached identifiers.
2. **Collection is graph-based.** Cookies and storage are cleared as a *set*, computed
   from the cross-site request graph, not cookie-by-cookie.
3. **CNAME-cloaked trackers are first-class.** A first-party subdomain that resolves via
   CNAME to a third party is treated as that third party for grouping and clearing.
4. **Partitioned by top-level site by default.** A third party cannot correlate across
   first-party sites.
5. **Farbling is deterministic per origin.** Spoofed values are stable within an origin
   (so sites work) but uncorrelated across origins.

## 2. Cookie storage model

### 2.1 Partitioning

Cookies are stored in `~/NetSurf/cookies/jar.jsonl`, keyed by the tuple:

```
(top-level-site, cookie-origin, name, path)
```

where `top-level-site` is the registrable domain of the first-party document
(`x/net/publicsuffix`) and `cookie-origin` is the origin that set the cookie. This is the
same partitioning applied to the cache and origin storage.

### 2.2 Cookie classes

| Class | Definition | On tracking-session reset |
|---|---|---|
| First-party persistent | Set by the top-level site, has an expiry | Kept (login/session continuity). |
| First-party session | Set by the top-level site, no expiry | Rotated/cleared. |
| Third-party | Set by any other origin in the graph | Cleared. |
| Cloaked third-party | Set by a host whose DNS CNAME chain leaves the first-party site | Cleared as its group. |
| Partitioned storage | `localStorage`/`sessionStorage`/IndexedDB per origin | Session cleared; persistent kept unless reset. |

### 2.3 The cookie jar API

```go
// storages/cookies
type Jar interface {
    Set(cookie Cookie) error
    Cookies(ctx context.Context, request Request) ([]Cookie, error)
    Delete(cookie Cookie) error
    ClearScope(scope Scope) error
}

type Scope struct {
    TopLevelSite string
    Origin       urls.Origin
    Session      SessionID // empty = all sessions for the partition
}
```

## 3. Cross-site request graph

The broker records every resource request as an edge between origins, including the DNS
resolution result.

```go
type DependencyGraph struct {
    // Nodes are origins and resolved endpoints.
    Nodes map[NodeID]*Node
    // Edges: From requested a resource on To, attributed to a first-party site.
    Edges []Edge
}

type Node struct {
    ID          NodeID
    Origin      urls.Origin
    Canonical   string   // DNS canonical name if different from Origin.Host
    Registrable string   // registrable domain of the canonical name (if resolvable)
    Kind        NodeKind // FirstParty, ThirdParty, CloakedThirdParty, TrackerGroup
}

type Edge struct {
    FromSite   string   // registrable domain of the top-level site
    From       NodeID
    To         NodeID
    Session    SessionID
    Kind       RequestKind // document, subresource, websocket, beacon, ...
}
```

### 3.1 CNAME-cloaking defense

1. `protocols/dns` returns the full CNAME chain, not just the final address.
2. If the canonical name's registrable domain differs from the requested host's
   registrable domain, the endpoint is classified `CloakedThirdParty`.
3. All first-party hosts that cloak to the same canonical registrable domain share a
   `TrackerGroup` node, so they are cleared together.
4. The group persists for the lifetime of the session graph and is recomputed as DNS
   results change.

Example: `metrics.example.com` is a CNAME to `collect.tracker.net`.

- Request to `metrics.example.com` creates a `CloakedThirdParty` node with canonical
  `collect.tracker.net`.
- Any cookie set under `metrics.example.com` is grouped with `collect.tracker.net`.
- On reset, all cookies for the group are deleted atomically, including cookies acquired
  through any other first-party subdomain that cloaks to the same tracker.

## 4. Tracking sessions

A tracking session is the unit of resettable identity.

```go
type TrackingSession struct {
    ID          SessionID
    TopLevel    string
    CreatedAt   time.Time
    Nodes       map[NodeID]struct{}
    CookieKeys  map[CookieKey]struct{}
    StorageNS   []string
    // Network-level identifiers that must also rotate:
    TLSResume   bool // per-session TLS session cache isolated
    CacheScope  string
}
```

### 4.1 Lifecycle

- A session is created on the first navigation to a site in a tab.
- All requests, cookies, storage writes, and cached identifiers are tagged with the
  session ID.
- **Reload / "New tracking session"** performs an atomic reset for the session:

  1. Recompute the dependency graph for the session.
  2. Delete all third-party and cloaked-third-party cookies for the graph's nodes.
  3. Delete first-party session cookies.
  4. Clear session-scoped origin storage.
  5. Rotate the session ID; isolate TLS session cache and any HTTP connection state.
  6. Optionally bypass the cache for this navigation (Reload) or retain the archive
     (New tracking session without refresh).

- An **isolated tab** owns its sessions only; clearing a session in one tab never affects
  another tab's session.

### 4.2 Atomicity

Collection and deletion happen in the storage broker as one transaction. If the process
crashes mid-reset, the journal replays to either the pre-reset or post-reset state; a
partial set is never left behind.

### 4.3 Interaction with the archive cache

- The archive cache is separate from the tracking session. Resetting a session does not
  delete the user's long-term archive unless the user chooses "clear archive for this
  site".
- Cached resources can carry server-set identifiers (e.g. `ETag`). On session reset,
  cached entries for the graph's origins are marked as **session-dirty** and revalidated or
  refetched with fresh validators, so a cached ETag cannot reconnect sessions.

## 5. Network privacy defaults

| Concern | Default |
|---|---|
| Transport | HTTPS-only; HTTP failures are not silently downgraded; mixed content blocked. |
| DNS | DNS-over-HTTPS with padding; no cookies to DoH endpoints; special-use names resolved locally. |
| TLS | TLS 1.3; 0-RTT disabled; per-site session cache; ECH where available; GREASE. |
| Referrer | `strict-origin-when-cross-origin`; `Referer` suppressed on cross-origin downgrades. |
| Headers | Normalized `Accept-Language`/`Accept`; no `From`; no unique `User-Agent`; `Sec-GPC: 1` and `DNT: 1`. |
| Cookies | Third-party blocked by default; partitioned; `SameSite` enforced; `__Host-`/`__Secure-` prefixes. |
| Cache/storage | Partitioned by top-level site. |
| WebRTC | No host-candidate leakage; mDNS candidates or none until permission. |

## 6. Farbling

`farbling/` provides deterministic spoofing of observable Web API values.

### 6.1 Seed and derivation

```go
type Profile struct {
    installSeed [32]byte // random per install, stored in ~/NetSurf/config/
    origin      string
    session     string
}

// Per-origin, per-session deterministic bytes.
func (profile *Profile) Noise(purpose string, index uint64) []byte {
    return hmacSHA256(profile.installSeed, profile.origin+"\x00"+profile.session+"\x00"+purpose, index)
}
```

Values are stable within an origin and session so that sites function, but differ across
origins so that fingerprints cannot be correlated.

### 6.2 Spoofing surfaces

- **Canvas / OffscreenCanvas**: per-pixel perturbation keyed by (origin, pixel index) for
  `getImageData`, `toDataURL`, `toBlob`; consistent across repeated reads.
- **WebGL / WebGL2**: spoofed vendor/renderer/parameters; noise in `readPixels`.
- **Audio**: inaudible sample perturbation for `AnalyserNode` and rendering.
- **Fonts**: fixed common font list; normalized metric probing; no unique font
  enumeration.
- **Navigator / Screen**: fixed plausible profile (Windows 11, Chrome/Edge), consistent
  `hardwareConcurrency`, `deviceMemory`, languages, and platform.
- **Media devices / battery / gamepad**: fixed minimal sets.
- **Timing**: coarsened `performance.now()`; no high-resolution leak.
- **Client Hints**: fixed low-entropy values; high-entropy hints disabled.

### 6.3 Farbling levels

| Level | Behavior |
|---|---|
| Off | Real values (for debugging; not recommended). |
| Standard | Farble high-entropy surfaces only. |
| Strict | Farble all surfaces and normalize aggressively. |

## 7. Privacy dashboard

The shell exposes a dashboard showing, per top-level site:

- the dependency graph of contacted origins, with cloaked third parties highlighted,
- cookies grouped by class and tracker group,
- actions: "New tracking session", "Clear this site", "Block this tracker group",
  "Clear archive for this site".

## 8. Decisions

- **D-PRIV-001:** Cookies are persistent in `~/NetSurf/cookies/` but session-scoped and
  partitioned; Reload creates a fresh tracking session.
- **D-PRIV-002:** Clearing is graph-based and atomic, grouping CNAME-cloaked third
  parties.
- **D-PRIV-003:** ETags and other cached identifiers are session-dirtied on reset.
- **D-PRIV-004:** Farbling is deterministic per origin+session via HMAC of an install
  seed.
- **D-PRIV-005:** DoH, HTTPS-only, third-party cookie blocking, and cache/storage
  partitioning are on by default.

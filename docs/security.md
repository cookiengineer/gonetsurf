# Security Model and Threat Model

## 1. Trust boundaries

| Component | Trust | Capabilities |
|---|---|---|
| Shell / UI | Trusted application | Windows, input, tab model. No parsing of untrusted content. |
| Renderer | **Untrusted** | Parses HTML/CSS, runs JS, lays out, paints. No sockets, no ambient FS. |
| Network broker | Trusted | Sockets, DNS, TLS, cookie jar, HTTP cache, origin storage. |
| Storage broker | Trusted | Partitioned origin storage and settings. |

All data crossing a boundary is untrusted until validated. The broker treats every
byte from the network, and every request from a renderer, as hostile.

## 2. Threat model

| Threat | Vector | Primary controls |
|---|---|---|
| Memory-safety or logic bug in HTML/CSS/JS/layout/paint | Malicious content | Process isolation, Linux sandbox, budgets, fuzzing. |
| Sandbox escape | Renderer exploit | Namespaces + seccomp + Landlock + capability drop + no-new-privs, min kernel surface. |
| Request smuggling | Malicious server/proxy | Strict framing, no `CL`+`TE`, no whitespace before `:`, reject obs-fold. |
| Response splitting / header injection | Malicious server, CRLF in values | Reject CR/LF in header names/values at the API boundary. |
| Cache poisoning | Conflicting headers, Vary confusion | Canonical effective URI, strict Vary, partition keys, framing rules. |
| DNS poisoning / leaks | On-path attacker, cleartext DNS | DoH/DoT, response ID + question matching, bounded resolver work. |
| Path traversal via filenames | `Content-Disposition`, cache mapping | Last-segment-only sanitization, reserved sentinels, no `..`/device names. |
| Resource-exhaustion DoS | DOM/CSS/image bombs | Node/rule/pixel/time budgets, watchdog termination, cgroup limits. |
| Fingerprinting / tracking | JS, headers, cookies, cache | Farbling, partitioning, tracking-session reset, normalized headers. |
| Cross-site data theft | Same-origin bypass, cache probing | Partitioned storage/cache, origin checks, no `document.domain` relaxation. |
| Malicious certificate | MITM | No bypass, HSTS, certificate verification errors are fatal. |

## 3. Renderer sandbox (Linux)

Implemented in `runtime/`:

1. **Namespaces:** user, network, mount, PID. The renderer has no network interfaces
   (`CLONE_NEWNET`) and cannot see the host filesystem (`CLONE_NEWNS`).
2. **No new privileges:** `PR_SET_NO_NEW_PRIVS`; setuid/setgid binaries cannot escalate.
3. **Capability drop:** all capabilities dropped.
4. **seccomp:** classic-BPF allow-list built with `x/sys/unix`, installed with thread
   synchronization so all Go runtime threads are covered.
5. **Landlock:** read-only access to system fonts and CA bundles; read/write access only
   to a per-renderer scratch directory.
6. **Resource limits:** `RLIMIT_AS`, `RLIMIT_CPU`, `RLIMIT_NOFILE`, `RLIMIT_FSIZE=0`,
   `RLIMIT_CORE=0`; cgroup v2 `memory.max`, `pids.max`, `cpu.max`.
7. **Watchdog:** the supervisor terminates a renderer that exceeds memory/time budgets or
   stops draining IPC.

The broker is the only process with sockets and the only writer to the cache/cookie
directories.

## 4. Input hardening per subsystem

| Subsystem | Hardening |
|---|---|
| `urls` | WHATWG parser only; no `net/url` on security paths; origin checks via parsed origin. |
| HTML | Node/attribute/depth budgets; `x/net/html` plus our own caps. |
| CSS | Rule/declaration/value budgets; bounded custom-property substitution; fuzzing. |
| HTTP | Header limits, framing precedence, smuggling corpus, no header injection. |
| DNS | Compression-pointer loop limits; ID + question matching; bounded retries. |
| Cache | Atomic writes; reserved sentinels; sanitized filenames; partition keys. |
| Images | Format sniffing, dimension/pixel budgets, decompression-bomb rejection. |
| JS | Per-task time budget, call-stack limit, interrupt on navigation, memory cap. |

## 5. Content security policy

- CSP is parsed and enforced by the renderer for document and subresource loading.
- `Content-Security-Policy` and the `<meta http-equiv>` form are both supported.
- The archive banner and all browser-injected UI are outside the page's CSP and cannot be
  spoofed or hidden by content.

## 6. Origin and storage isolation

- Same-origin checks use `urls.Origin`, never string comparison.
- Storage and cache are partitioned by top-level site.
- `document.domain` relaxation is disabled unless explicitly enabled by the user.
- The cache cannot be used as a cross-origin oracle: partition keys are part of lookup.

## 7. Crash and error handling

- A panic in a parser, layout pass, or script task is recovered at the renderer task
  boundary and reported as a page error, never propagated to the shell or broker.
- A renderer crash shows a recoverable error page; the tab can reload.
- Broker errors never crash the shell; they surface as network error pages.
- Security-relevant failures (certificate errors, mixed content, blocked requests) are
  logged without leaking browsing history.

## 8. Verification

- **Fuzzing:** native Go fuzz targets for URL, HTML wrapper, CSS tokenizer/parser, HTTP
  framing, DNS, SVG, and the cache path mapper.
- **Escape tests:** attempts to reach the network or host filesystem from the renderer
  must fail and be caught in CI.
- **Smuggling/splitting corpus:** a regression corpus is run on every change to the HTTP
  parser.
- **Budget tests:** DOM/CSS/image bombs must be rejected before exhausting memory.

## 9. Decisions

- **D-SEC-001:** The broker is the only process with network and cache/cookie filesystem
  access.
- **D-SEC-002:** Renderers are sandboxed with namespaces, seccomp, Landlock, rlimits, and
  cgroups.
- **D-SEC-003:** No certificate bypass exists in the codebase.
- **D-SEC-004:** All security decisions use `urls` (WHATWG), never `net/url`.

# Delta for self-update

## MODIFIED Requirements

### Requirement: GitHub Release Detection

The system MUST query `https://api.github.com/repos/JhnFrankz/upp/releases/latest` over HTTPS with ~10s dial/read timeouts and MUST use a 24h TTL cache (bounds the 60 req/h unauthenticated limit). A fresh cache MUST be reused without network. API failure on `upp self-update` MUST report a clear error and exit non-zero. Release detection runs ONLY as part of `upp self-update`: no other command MAY trigger release detection or any self-update network call, and self-update remains manual-only.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Fresh cache | Cache < 24h old | Detection | No network call, cached result used |
| API failure (command) | API unreachable | `upp self-update` | Clear error, exit non-zero |
| Stale cache | Cache > 24h old | Detection | Re-fetch, cache refreshed |
| No hint-path detection | Any command other than `upp self-update` runs | Execution | Zero release-detection network calls from that command |

(Previously: an API failure on the opt-in hint path (`upp check`) had to stay silent with the exit code unchanged; the hint path no longer exists and detection is exclusive to `upp self-update`.)

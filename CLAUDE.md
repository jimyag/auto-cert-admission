# CLAUDE.md

Agent guidance for the auto-cert-webhook project.

## Project Overview

A Go library/framework for building Kubernetes admission webhooks with automatic TLS
certificate management, leader election, and Prometheus metrics.

Consumers implement the `Admission` interface (`Configure() Config` + `Webhooks() []Hook`)
and call `webhook.Run(impl)` — everything else is handled by the framework.

## Public API Surface

Only these symbols are exported at the package root (`github.com/jimyag/auto-cert-webhook`):

- `Run(Admission) error` / `RunWithContext(ctx, Admission) error` — entry points
- `Admission` interface — what consumers implement
- `Config` struct — webhook configuration (code > env > defaults)
- `Hook` struct — single webhook endpoint registration
- `HookType` (`Mutating` / `Validating`)
- `AdmitFunc` — `func(admissionv1.AdmissionReview) *admissionv1.AdmissionResponse`
- Response helpers: `Allowed()`, `AllowedWithMessage()`, `Denied()`, `DeniedWithReason()`,
  `Errored()`, `ErroredWithCode()`, `PatchResponse()`, `PatchResponseFromRaw()`,
  `PatchResponseFromPatches()`

Do not add new exported symbols without a clear consumer need — the surface is intentionally
small.

## Internal Package Responsibilities

```
internal/
  certmanager/   Leader-only. Rotates CA and serving certs via openshift/library-go.
                 Polls on a configurable interval (default 1m). Creates Secrets if absent.
  certprovider/  All replicas. Watches the cert Secret via informer (event-driven, no polling).
                 Stores current *tls.Certificate in atomic.Pointer for lock-free hot-reload.
  cabundle/      Leader-only. Patches caBundle into MutatingWebhookConfiguration /
                 ValidatingWebhookConfiguration. Driven by certmanager's sync cycle.
  leaderelection/ Wraps client-go leader election. Uses LeaseLock on coordination.k8s.io/v1.
                 ReleaseOnCancel=true. Retries automatically if election exits without cancel
                 (PR #3: leadership recovery).
  metrics/       All replicas. Prometheus metrics for cert expiry and leader state.
                 leader_observer.go watches the Lease via informer and updates leader metrics
                 (PR #4: leader election monitoring).
  server/        All replicas. TLS server using certprovider.GetCertificate as the TLS
                 GetCertificate callback — no file watching, zero-downtime cert reload.
```

## Configuration Priority Model

`code > environment variables (ACW_* prefix) > struct tag defaults`

Implemented in `run.go:applyEnvConfig`. The trick: save a deep copy of the user struct
before calling `envconfig.Process`, then restore any non-zero / non-nil user values
afterwards. Do not bypass this by reading env vars directly in other packages.

## Certificate Duration Constraints

Validated at startup in `run.go:validateCertDurations`:

- `CARefresh < CAValidity` (default: 24h < 48h)
- `CertRefresh < CertValidity` (default: 12h < 24h)

Both must be positive. Tests that set durations must respect these constraints.

## Key Naming Convention

`Config.Name` drives all derived resource names — and is the single most important
constraint for deployers:

| Derived from `Config.Name` | Default |
|---|---|
| CA Secret | `<Name>-ca` |
| Cert Secret | `<Name>-cert` |
| CA Bundle ConfigMap | `<Name>-ca-bundle` |
| Leader Election Lease | `<Name>-leader` |
| MutatingWebhookConfiguration | must be `<Name>` (not derived, must match) |
| ValidatingWebhookConfiguration | must be `<Name>` (not derived, must match) |

The caBundle syncer looks up WebhookConfigurations by `Config.Name` — if the names
don't match the caBundle will never be populated.

## Leader vs All-Replica Components

```
All replicas:
  certprovider    — watches cert Secret, hot-reloads tls.Certificate
  server          — TLS admission server
  metrics server  — Prometheus /metrics
  leader_observer — watches Lease, updates leader metrics

Leader only:
  certmanager     — rotates CA + serving cert, writes CA bundle ConfigMap
  cabundle.Syncer — patches caBundle into WebhookConfiguration
```

In single-replica mode (`LeaderElection=false`), the framework skips leader election
and runs leader components directly. It also reports the current instance as leader in
metrics so dashboards and alerts work unchanged.

## Error Handling Model

All goroutines send fatal errors to a shared buffered `errCh chan error` (capacity 7,
see `run.go`). Context-cancellation errors are silently discarded. The main goroutine
exits on the first real error received.

Do not return transient errors (e.g., a single failed cert sync tick) to `errCh` —
those should be logged and retried in the next tick.

## Logging

Uses `k8s.io/klog/v2`. Follow existing patterns:

- `klog.Infof` for state transitions (became leader, cert reloaded, etc.)
- `klog.V(4).Infof` for verbose/debug (sync ticks, cache hits)
- `klog.Warningf` for recoverable situations (initial load failed, will retry)
- `klog.Errorf` for non-fatal errors that need attention

Do not use `fmt.Println` or `log` from stdlib.

## Key Dependencies

| Package | Role |
|---|---|
| `github.com/openshift/library-go` | CA rotation (`certrotation.RotatedSigningCASecret`, `RotatedSelfSignedCertKeySecret`, `CABundleConfigMap`) |
| `k8s.io/client-go/tools/leaderelection` | Leader election via LeaseLock |
| `k8s.io/client-go/informers` | Event-driven Secret/Lease watching (no polling) |
| `github.com/kelseyhightower/envconfig` | ACW_* env var parsing |
| `github.com/appscode/jsonpatch` | JSON Patch generation for mutating responses |
| `github.com/prometheus/client_golang` | Metrics |
| `k8s.io/klog/v2` | Logging |

When upgrading `openshift/library-go`, verify that `certrotation` API is unchanged —
it has broken in past minor versions.

## RBAC Permissions

When adding features that touch new API resources or verbs, update ALL of the following
in sync:

- `README.md` — the "Required RBAC" section (canonical human-readable reference)
- `examples/pod-mutating/deploy/02-rbac.yaml`
- `examples/pod-validating/deploy/02-rbac.yaml`

### Current required rules and rationale

| apiGroup | resource | verbs | reason |
|---|---|---|---|
| `""` | `secrets`, `configmaps` | get, list, watch, create, update | CA/serving cert storage; CA bundle ConfigMap |
| `coordination.k8s.io` | `leases` | get, list, watch, create, update | Leader election (create/update to hold lease; list/watch for informer and leader metrics — PR #4) |
| `admissionregistration.k8s.io` | `mutatingwebhookconfigurations`, `validatingwebhookconfigurations` | get, update, patch | Leader patches caBundle so API server trusts the webhook TLS cert |
| `""` | `events` | create, patch | Leader election and cert rotation emit Kubernetes events |

Note: `list` and `watch` on `leases` are required by the `informers.SharedInformerFactory`
used in `leader_observer.go`. Without them the informer cannot start, blocking leader
metrics (PR #4) and leadership recovery (PR #3).

## What NOT to Do

- Do not use `time.Sleep` / polling for synchronization — use informers or channels.
- Do not add file watching (e.g., `inotify`, `fsnotify`) — cert hot-reload is done via
  the Secret informer in `certprovider`.
- Do not read env vars directly outside `run.go:applyEnvConfig` — all config flows
  through `Config`.
- Do not return context-cancellation errors from goroutines to `errCh` — check
  `ctx.Err()` before sending (see `reportAsyncError`).
- Do not export new symbols from `internal/` packages — they are framework-internal.

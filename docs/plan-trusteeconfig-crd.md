# Plan: Introduce High-Level TrusteeConfig CRD

## Context

This helm-based operator needs a two-level CRD hierarchy: a high-level `TrusteeConfig` CR (profile-driven) that translates into the detailed `Trustee` CR (which drives the Helm chart). This gives users a simple interface instead of configuring dozens of Helm values.

**Phase 1 (this plan):** Implement a Permissive TrusteeConfig profile only — insecure HTTP, no token verification. This is the simplest useful configuration. The Helm chart already hardcodes Permissive defaults (`insecure_http = true`, `authorization_mode = "AuthenticatedAuthorization"`), so no Helm chart changes are needed.

**Future phases:** HTTPS support, attestation token verification, IBM SE, and a Restricted profile will be added once the upstream trustee helm charts support them. At that point, the Helm values and templates will need to be made configurable.

## Step 1: Fix `cleanEmpty` bool stripping

**File:** `internal/helm/values.go`

Remove the `case bool` clause that strips `false` bools. Explicit `false` in Helm values is always safe and needed for future phases.

## Step 2: Create TrusteeConfig CRD

**New file:** `api/v1alpha1/trusteeconfig_types.go`

```go
type ProfileType string  // Permissive (Restricted reserved for future)

type TrusteeConfigSpec struct {
    Profile        ProfileType        `json:"profile"`
    KbsServiceType corev1.ServiceType `json:"kbsServiceType,omitempty"`
}

type TrusteeConfigStatus struct {
    Conditions         []metav1.Condition
    ObservedGeneration int64
    TrusteeRef         *corev1.ObjectReference
}
```

Printer columns: Profile, Ready, Age.

Register `TrusteeConfig` and `TrusteeConfigList` in `groupversion_info.go`.

## Step 3: Create TrusteeConfig controller

**New file:** `internal/controller/trusteeconfig_controller.go`

Translation logic for Permissive profile:

| Field | Permissive |
|---|---|
| `logLevel` | `debug` |

Additional mappings:
- `kbsServiceType=NodePort` -> `nodePort.enabled=true`
- `kbsServiceType=LoadBalancer` -> `kbs.service.exposeLoadBalancer=true`

Controller creates/updates a `Trustee` CR with owner reference. Uses `Owns(&Trustee{})` to propagate status back. Compares specs with `DeepEqual` to avoid infinite reconcile loops.

**File:** `cmd/main.go` — Register `TrusteeConfigReconciler`

## Step 4: Project scaffolding

- Update `PROJECT` file with new TrusteeConfig resource
- Update `config/crd/kustomization.yaml` to include new CRD
- Create RBAC roles (`trusteeconfig_editor_role.yaml`, `trusteeconfig_viewer_role.yaml`)
- Sample CR: `config/samples/trustee_v1alpha1_trusteeconfig.yaml`
- Run `make generate manifests` for deepcopy and CRD YAML generation

## Verification

1. `make generate manifests` succeeds
2. `make build` compiles
3. `make test` passes (envtest-based controller tests)

## Future work (deferred)

The following will be added once upstream trustee helm charts support them:
- **HTTPS:** TLS certificates, `insecureHttp: false`, TLS profile config, secret volume mounts, HTTPS health probes
- **Attestation token verification:** Token verification secrets, `insecureHeaderJwk: false`, CoCo token type
- **IBM SE:** Credentials directory and node name propagation
- **Restricted profile:** Combines HTTPS + token verification + stricter defaults (`DenyAll` authorization, `workerCount: 4`, `logLevel: info`)
- **Helm chart changes:** Make `insecure_http`, `authorization_mode`, `attestation_token` settings value-driven in `kbs-config.toml.template`

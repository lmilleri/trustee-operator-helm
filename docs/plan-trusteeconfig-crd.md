# Plan: Introduce High-Level TrusteeConfig CRD

## Context

The existing trustee-operator at `/home/lmilleri/git/trustee-operator` has a two-level CRD hierarchy: a high-level `TrusteeConfig` (profile-driven: Permissive/Restricted, HTTPS, TLS, IBM SE) translates into a detailed `KbsConfig`. This helm-based operator needs the same pattern: a simple, profile-driven TrusteeConfig CR that translates into the detailed Trustee CR (which drives the Helm chart). This gives users a simple interface (choose Permissive/Restricted profile + a few options) instead of configuring dozens of Helm values.

## Reference files

- Existing trustee-operator TrusteeConfig types: `/home/lmilleri/git/trustee-operator/api/v1alpha1/kbsconfig_types.go`
- Existing trustee-operator controller: `/home/lmilleri/git/trustee-operator/internal/controller/trusteeconfig_controller.go`
- Restricted config template example: `/home/lmilleri/git/trustee-operator/config/templates/kbs-config-restricted.toml`

## Step 1: Fix `cleanEmpty` bool stripping

**File:** `internal/helm/values.go`

Remove the `case bool` clause (lines 71-73). Currently strips `false` bools, which would prevent `insecureHttp: false` from reaching Helm. Explicit `false` in Helm values is always safe.

## Step 2: Extend `TrusteeSpec` with new types

**File:** `api/v1alpha1/trustee_types.go`

Add new types:

```go
type TLSProfileConfig struct {
    Profile    string   `json:"profile,omitempty"`     // old/intermediate/modern/custom
    MinVersion string   `json:"minVersion,omitempty"`
    MaxVersion string   `json:"maxVersion,omitempty"`
    Ciphers    []string `json:"ciphers,omitempty"`
    Groups     []string `json:"groups,omitempty"`
}

type HttpServerConfig struct {
    InsecureHttp *bool             `json:"insecureHttp,omitempty"`
    WorkerCount  *int32            `json:"workerCount,omitempty"`
    TLS          *TLSProfileConfig `json:"tls,omitempty"`
}

type AttestationTokenConfig struct {
    InsecureHeaderJwk *bool  `json:"insecureHeaderJwk,omitempty"`
    Type              string `json:"type,omitempty"`
}

type HttpsSecretSpec struct {
    ExistingTlsSecret string `json:"existingTlsSecret,omitempty"`
}

type AttestationTokenVerificationSecretSpec struct {
    ExistingTlsSecret string `json:"existingTlsSecret,omitempty"`
}
```

Extend existing types:
- Add `AuthorizationMode string` to `KBSAdminConfig`
- Add `HttpServer`, `AttestationToken` fields to `KBSConfig`
- Add `Https`, `AttestationTokenVerification` fields to `KBSSpec`

## Step 3: Update Helm values and templates

**File:** `helm-charts/trustee/values.yaml` — Add under `kbs.config`:
- `httpServer.insecureHttp: true`, `httpServer.workerCount: null`, `httpServer.tls.*`
- `admin.authorizationMode: AuthenticatedAuthorization`
- `attestationToken.insecureHeaderJwk: null`, `attestationToken.type: ""`

Add under `kbs`:
- `https.existingTlsSecret: ""`
- `attestationTokenVerification.existingTlsSecret: ""`

**File:** `helm-charts/trustee/files/kbs-config.toml.template` — Make `insecure_http`, `private_key`/`certificate`, TLS profile, `authorization_mode`, admin JWT sections, and `attestation_token` settings all value-driven instead of hardcoded. Key logic:
- `insecure_http` reads from `kbs.config.httpServer.insecureHttp` (default `true`)
- When `insecureHttp=false`, emit `private_key`/`certificate` paths pointing to `/etc/https-certs/tls.key` and `tls.crt`
- TLS profile block rendered only when `insecureHttp=false` and `tls.profile` is set
- `admin.authentication` and `admin.authorization` sections emitted only when `authorizationMode != "DenyAll"`

**File:** `helm-charts/trustee/templates/kbs-deployment.yaml` — Add conditional volume/volumeMount entries:
- `https-certs` volume from `kbs.https.existingTlsSecret` mounted at `/etc/https-certs`
- `attestation-certs` volume from `kbs.attestationTokenVerification.existingTlsSecret` mounted at `/etc/attestation-certs`
- Health probes: add `scheme: HTTPS` when `insecureHttp=false`

## Step 4: Create TrusteeConfig CRD

**New file:** `api/v1alpha1/trusteeconfig_types.go`

```go
type ProfileType string  // Permissive | Restricted

type TrusteeConfigSpec struct {
    Profile                          ProfileType
    HttpsSpec                        TrusteeConfigHttpsSpec                        // TlsSecretName
    AttestationTokenVerificationSpec TrusteeConfigAttestationTokenVerificationSpec // TlsSecretName
    IbmSE                           *TrusteeConfigIbmSESpec                       // CredsDir, NodeName
    KbsServiceType                  corev1.ServiceType
    TlsConfig                       *TrusteeConfigTlsConfig                       // profile, versions, ciphers, groups
}

type TrusteeConfigStatus struct {
    Conditions         []metav1.Condition
    ObservedGeneration int64
    TrusteeRef         *corev1.ObjectReference
}
```

Printer columns: Profile, Ready, Age.

## Step 5: Create TrusteeConfig controller

**New file:** `internal/controller/trusteeconfig_controller.go`

Translation logic per profile:

| Field | Permissive | Restricted |
|---|---|---|
| `logLevel` | `debug` | `info` |
| `kbs.config.httpServer.insecureHttp` | `true` | `false` |
| `kbs.config.httpServer.workerCount` | — | `4` |
| `kbs.config.admin.authorizationMode` | `AuthenticatedAuthorization` | `DenyAll` |
| `kbs.config.attestationToken.insecureHeaderJwk` | `true` | `false` |
| `kbs.config.attestationToken.type` | — | `CoCo` |

Additional mappings:
- `httpsSpec.tlsSecretName` → `kbs.https.existingTlsSecret`
- `attestationTokenVerificationSpec.tlsSecretName` → `kbs.attestationTokenVerification.existingTlsSecret`
- `ibmSE.credsDir/nodeName` → `as.verifier.se.credsDir/nodeName`
- `kbsServiceType=NodePort` → `nodePort.enabled=true`
- `kbsServiceType=LoadBalancer` → `kbs.service.exposeLoadBalancer=true`
- `tlsConfig.*` → `kbs.config.httpServer.tls.*`

Controller creates/updates a `Trustee` CR with owner reference. Uses `Owns(&Trustee{})` to propagate status back. Compares specs with `DeepEqual` to avoid infinite reconcile loops.

**File:** `cmd/main.go` — Register `TrusteeConfigReconciler`

## Step 6: Project scaffolding

- Update `PROJECT` file with new TrusteeConfig resource
- Update `config/crd/kustomization.yaml` to include new CRD
- Create RBAC roles (`trusteeconfig_editor_role.yaml`, `trusteeconfig_viewer_role.yaml`)
- Run `make generate manifests` for deepcopy and CRD YAML generation

## Step 7: Samples and tests

Sample CRs:
- `config/samples/trustee_v1alpha1_trusteeconfig.yaml` (Permissive)
- `config/samples/trustee_v1alpha1_trusteeconfig_restricted.yaml` (Restricted with HTTPS + TLS)

Unit tests in `internal/controller/trusteeconfig_controller_test.go`:
- Permissive profile → correct Trustee spec values
- Restricted profile → correct Trustee spec values
- HTTPS/attestation secret propagation
- IBM SE propagation
- Service type mapping (NodePort, LoadBalancer)
- TLS config propagation
- Owner reference set correctly
- Update idempotency (no infinite loops)

## Verification

1. `make generate manifests` succeeds
2. `make build` compiles
3. `make test` passes (envtest-based controller tests)
4. `helm template` with default values produces same output as before (backward compatible)
5. `helm template` with `kbs.config.httpServer.insecureHttp=false` renders HTTPS config correctly

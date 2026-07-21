# Trustee Operator (Helm-based)

A Kubernetes operator for deploying and managing [CoCo Trustee](https://github.com/confidential-containers/trustee) — the Key Broker Service (KBS), Attestation Service (AS), and Reference Value Provider Service (RVPS) for confidential containers.

## Description

This operator uses an embedded Helm chart to deploy and lifecycle-manage a full Trustee stack. Users create a `Trustee` custom resource describing the desired deployment, and the operator translates the CR spec into Helm values, installs or upgrades the release, and continuously monitors the health of KBS, AS, and RVPS deployments.

The operator supports:
- **LocalFs, LocalJson, Postgres, and Memory** storage backends
- **Bundled or external PostgreSQL** (via the Bitnami subchart)
- **Ephemeral or pre-created** cryptographic key material
- **Ingress, NodePort, and LoadBalancer** exposure for KBS
- **NVIDIA, Intel DCAP, and IBM Secure Execution** attestation verifiers
- Per-component replica counts, images, resources, tolerations, affinities, and node selectors

## Getting Started

### Prerequisites
- Go 1.26+
- Docker 17.03+
- kubectl v1.28+
- Access to a Kubernetes v1.28+ cluster

### Deploy on a cluster

Build and push the operator image:

```sh
make docker-build docker-push IMG=<some-registry>/trustee-operator:tag
```

Install the CRDs:

```sh
make install
```

Deploy the operator:

```sh
make deploy IMG=<some-registry>/trustee-operator:tag
```

> If you encounter RBAC errors, you may need cluster-admin privileges.

### Create a Trustee instance

Apply one of the sample CRs:

```sh
kubectl apply -k config/samples/
```

Minimal example:

```yaml
apiVersion: trustee.confidentialcontainers.org/v1alpha1
kind: Trustee
metadata:
  name: trustee-sample
spec:
  logLevel: info
  storageBackend:
    type: LocalFs
  secrets:
    useEphemeralGeneratedKeys: true
```

### Uninstall

```sh
kubectl delete -k config/samples/
make uninstall
make undeploy
```

## Project Distribution

### YAML bundle

Build the all-in-one installer:

```sh
make build-installer IMG=<some-registry>/trustee-operator:tag
```

This generates `dist/install.yaml`. Users can install with:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/trustee-operator/<tag>/dist/install.yaml
```

## Contributing

1. Fork the repository and create a feature branch.
2. Make your changes and ensure `make lint test` passes.
3. Submit a pull request.

Run `make help` for the full list of available targets.

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html).

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

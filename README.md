# cert-manager-webhook-poweradmin

[![CI](https://github.com/poweradmin/cert-manager-webhook-poweradmin/actions/workflows/test.yaml/badge.svg)](https://github.com/poweradmin/cert-manager-webhook-poweradmin/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/poweradmin/cert-manager-webhook-poweradmin)](https://goreportcard.com/report/github.com/poweradmin/cert-manager-webhook-poweradmin)
[![GitHub release](https://img.shields.io/github/v/release/poweradmin/cert-manager-webhook-poweradmin)](https://github.com/poweradmin/cert-manager-webhook-poweradmin/releases)
[![License](https://img.shields.io/github/license/poweradmin/cert-manager-webhook-poweradmin)](LICENSE)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/poweradmin-webhook)](https://artifacthub.io/packages/helm/poweradmin-webhook/poweradmin-webhook)

A [cert-manager](https://cert-manager.io/) webhook for [PowerAdmin](https://www.poweradmin.org/) DNS provider, enabling DNS-01 ACME challenges for automated certificate issuance (e.g., Let's Encrypt) in Kubernetes.

## Prerequisites

- Kubernetes 1.25+
- [cert-manager](https://cert-manager.io/docs/installation/) 1.0+
- [Helm](https://helm.sh/) 3.0+
- PowerAdmin instance with API access enabled

## Container Images

Multi-platform images (amd64/arm64) are published on each release:

- [`ghcr.io/poweradmin/cert-manager-webhook-poweradmin`](https://ghcr.io/poweradmin/cert-manager-webhook-poweradmin)
- [`docker.io/poweradmin/cert-manager-webhook-poweradmin`](https://hub.docker.com/r/poweradmin/cert-manager-webhook-poweradmin)

## Installation

### Using Helm (OCI Registry)

```bash
helm install poweradmin-webhook oci://ghcr.io/poweradmin/charts/poweradmin-webhook \
  --namespace cert-manager \
  --set groupName=acme.yourdomain.com
```

### Using Helm (Local Chart)

```bash
helm install poweradmin-webhook deploy/poweradmin-webhook \
  --namespace cert-manager \
  --set groupName=acme.yourdomain.com
```

> **Important:** Set `groupName` to a unique domain you own. This value must match what you configure in your Issuer/ClusterIssuer.

### Uninstall

```bash
helm uninstall poweradmin-webhook --namespace cert-manager
```

## Configuration

### 1. Create API Key Secret

Create a Kubernetes Secret containing your PowerAdmin API key:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: poweradmin-api-key
  namespace: cert-manager
type: Opaque
stringData:
  api-key: "pwa_your_api_key_here"
```

### 2. Configure ClusterIssuer

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-account-key
    solvers:
    - dns01:
        webhook:
          groupName: acme.yourdomain.com
          solverName: poweradmin
          config:
            serverURL: "https://poweradmin.example.com"
            apiKeySecretRef:
              name: poweradmin-api-key
              key: api-key
            apiVersion: "v2"     # optional, default "v2"; also supports "v1"
            ttl: 120             # optional, default 120
            insecure: false      # optional, default false
```

### 3. Request a Certificate

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-cert
  namespace: default
spec:
  secretName: example-cert-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - example.com
  - "*.example.com"
```

## Configuration Options

| Field                  | Required | Default | Description                               |
|------------------------|----------|---------|-------------------------------------------|
| `serverURL`            | Yes      | -       | Base URL of the PowerAdmin instance       |
| `apiKeySecretRef.name` | Yes      | -       | Name of the Secret containing the API key |
| `apiKeySecretRef.key`  | Yes      | -       | Key within the Secret                     |
| `apiVersion`           | No       | `"v2"`  | PowerAdmin API version (`"v1"` or `"v2"`) |
| `ttl`                  | No       | `120`   | TTL for TXT records in seconds            |
| `insecure`             | No       | `false` | Skip TLS verification for PowerAdmin API  |

## Development

### Build

```bash
# Build the binary
go build .

# Build the Docker image
make build
```

### Test

```bash
# Run unit tests
make test-unit

# Run conformance tests (requires envtest)
make test
```

## Sponsors

<a href="https://menzel-it.net/de/">
  <img src=".github/menzel_it_logo.svg" alt="Menzel IT GmbH" height="60">
</a>

We thank [Menzel IT GmbH](https://menzel-it.net/de/) for their support of this project.

## License

Apache License 2.0 - see [LICENSE](LICENSE) file.

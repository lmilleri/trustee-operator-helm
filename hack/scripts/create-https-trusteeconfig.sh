#!/bin/bash
#
# create-https-trusteeconfig.sh
# Creates a TrusteeConfig (Permissive profile) with native HTTPS on the KBS
# endpoint for the Helm-based trustee-operator
# (group: trustee.confidentialcontainers.org/v1alpha1).
#
# Adapted from the non-helm trustee-operator's create-restrictive-trusteeconfig.sh.
# Differences for this operator:
#   - apiVersion is trustee.confidentialcontainers.org/v1alpha1
#   - profileType only supports "Permissive" (Restricted is not yet implemented)
#   - attestationTokenVerificationSpec and tlsConfig are not supported and are omitted
#   - the KBS Service is named "<config-name>-kbs" (not "kbs-service")
#

set -e

# Configuration
NAMESPACE="${NAMESPACE:-trustee-operator-system}"
CONFIG_NAME="${CONFIG_NAME:-trusteeconfig}"
HTTPS_SECRET_NAME="${HTTPS_SECRET_NAME:-trustee-tls-secret}"
SERVICE_TYPE="${SERVICE_TYPE:-ClusterIP}"
CERT_DAYS="${CERT_DAYS:-365}"
# Optional extra Subject Alternative Name (e.g. an external Route/Ingress host).
EXTRA_SAN="${EXTRA_SAN:-}"

# KBS Service name as rendered by the chart (coco-trustee.names.kbs => <fullname>-kbs).
# When CONFIG_NAME contains "trustee", fullname == CONFIG_NAME; otherwise the
# chart appends "-trustee". The default name below covers the common case.
KBS_SVC="${CONFIG_NAME}-kbs"

# Temporary files
TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Creates a Permissive TrusteeConfig with native HTTPS on the KBS endpoint,
for the Helm-based trustee-operator.

Options:
    -n, --namespace NAME        Kubernetes namespace (default: trustee-operator-system)
    -c, --config-name NAME      TrusteeConfig name (default: trusteeconfig)
    --https-secret NAME         HTTPS TLS secret name (default: trustee-tls-secret)
    --service-type TYPE         KBS service type (default: ClusterIP)
    --cert-days DAYS            Certificate validity in days (default: 365)
    --extra-san HOST            Additional DNS SAN for the cert (e.g. external Route host)
    --skip-secrets              Skip creating the TLS secret (assume it exists)
    -h, --help                  Show this help message

Environment Variables:
    NAMESPACE, CONFIG_NAME, HTTPS_SECRET_NAME, SERVICE_TYPE, CERT_DAYS, EXTRA_SAN

Example:
    $0
    $0 --namespace my-namespace --config-name my-trustee
    $0 --service-type LoadBalancer --extra-san kbs.example.com
    $0 --skip-secrets

EOF
}

# Parse arguments
SKIP_SECRETS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -c|--config-name)
            CONFIG_NAME="$2"
            KBS_SVC="${CONFIG_NAME}-kbs"
            shift 2
            ;;
        --https-secret)
            HTTPS_SECRET_NAME="$2"
            shift 2
            ;;
        --service-type)
            SERVICE_TYPE="$2"
            shift 2
            ;;
        --cert-days)
            CERT_DAYS="$2"
            shift 2
            ;;
        --extra-san)
            EXTRA_SAN="$2"
            shift 2
            ;;
        --skip-secrets)
            SKIP_SECRETS=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

log_info "Configuration:"
log_info "  Namespace: ${NAMESPACE}"
log_info "  Config Name: ${CONFIG_NAME}"
log_info "  HTTPS Secret: ${HTTPS_SECRET_NAME}"
log_info "  KBS Service: ${KBS_SVC}"
log_info "  Service Type: ${SERVICE_TYPE}"
log_info "  Certificate Days: ${CERT_DAYS}"
log_info "  Extra SAN: ${EXTRA_SAN:-<none>}"
log_info "  Skip Secrets: ${SKIP_SECRETS}"
echo

# Check if namespace exists
if ! kubectl get namespace "${NAMESPACE}" &>/dev/null; then
    log_error "Namespace '${NAMESPACE}' does not exist"
    exit 1
fi

if [ "${SKIP_SECRETS}" = false ]; then
    # Step 1: Create OpenSSL configuration for the HTTPS certificate
    log_info "Creating OpenSSL configuration..."
    cat > "${TEMP_DIR}/kbs-service-openssl.conf" <<EOF
[req]
default_bits       = 2048
distinguished_name = req_distinguished_name
req_extensions     = req_ext
x509_extensions    = v3_req

[req_distinguished_name]

[req_ext]
subjectAltName = @alt_names
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth

[v3_req]
subjectAltName = @alt_names
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth

[alt_names]
DNS.1 = ${KBS_SVC}
DNS.2 = ${KBS_SVC}.${NAMESPACE}.svc
DNS.3 = ${KBS_SVC}.${NAMESPACE}.svc.cluster.local
EOF
    if [ -n "${EXTRA_SAN}" ]; then
        echo "DNS.4 = ${EXTRA_SAN}" >> "${TEMP_DIR}/kbs-service-openssl.conf"
    fi

    # Step 2: Generate the HTTPS certificate and create the secret
    log_info "Generating HTTPS certificate (RSA 2048-bit, ${CERT_DAYS} days validity)..."
    openssl req -x509 -nodes -days "${CERT_DAYS}" -newkey rsa:2048 \
        -keyout "${TEMP_DIR}/tls.key" -out "${TEMP_DIR}/tls.crt" \
        -config "${TEMP_DIR}/kbs-service-openssl.conf" -passin pass: \
        -subj "/C=US/ST=MA/L=Boston/O=Red Hat/OU=Development/CN=${KBS_SVC}" 2>/dev/null

    # Check if the secret already exists
    if kubectl get secret "${HTTPS_SECRET_NAME}" -n "${NAMESPACE}" &>/dev/null; then
        log_warn "Secret '${HTTPS_SECRET_NAME}' already exists, deleting..."
        kubectl delete secret "${HTTPS_SECRET_NAME}" -n "${NAMESPACE}"
    fi

    log_info "Creating Kubernetes TLS secret for HTTPS: ${HTTPS_SECRET_NAME}"
    kubectl create secret tls "${HTTPS_SECRET_NAME}" \
        --cert="${TEMP_DIR}/tls.crt" \
        --key="${TEMP_DIR}/tls.key" \
        -n "${NAMESPACE}"

    log_info "Secret created successfully"
else
    log_info "Skipping secret creation (--skip-secrets specified)"

    # Verify the secret exists
    if ! kubectl get secret "${HTTPS_SECRET_NAME}" -n "${NAMESPACE}" &>/dev/null; then
        log_error "HTTPS secret '${HTTPS_SECRET_NAME}' does not exist"
        exit 1
    fi

    log_info "Verified existing secret"
fi

# Step 3: Create the TrusteeConfig CR
log_info "Creating TrusteeConfig: ${CONFIG_NAME}"

kubectl apply -f - <<EOF
apiVersion: trustee.confidentialcontainers.org/v1alpha1
kind: TrusteeConfig
metadata:
  name: ${CONFIG_NAME}
  namespace: ${NAMESPACE}
spec:
  profileType: Permissive
  kbsServiceType: ${SERVICE_TYPE}
  httpsSpec:
    tlsSecretName: ${HTTPS_SECRET_NAME}
EOF

echo
log_info "✓ TrusteeConfig '${CONFIG_NAME}' created successfully in namespace '${NAMESPACE}'"
log_info ""
log_info "Next steps:"
log_info "  1. Wait for the operator to reconcile: kubectl wait --for=condition=Ready trusteeconfig/${CONFIG_NAME} -n ${NAMESPACE} --timeout=5m"
log_info "  2. Check status: kubectl get trusteeconfig ${CONFIG_NAME} -n ${NAMESPACE} -o yaml"
log_info "  3. View resources: kubectl get trusteeconfig,trustee -n ${NAMESPACE}"
log_info "  4. Check the KBS service: kubectl get svc -n ${NAMESPACE} ${KBS_SVC}"

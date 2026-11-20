#!/bin/bash
#
# Helper script to interact with KBS via kbs-client.
# Extracts the admin token from the bootstrap secret and
# port-forwards to the KBS service.

set -euo pipefail

usage() {
    cat <<EOF
Usage: $0 [release-name] <command> [args...]

The script automatically handles admin token extraction and port-forwarding.

If release-name is omitted, it is auto-detected from the
"<release>-bootstrap-user-keys" secret in the namespace.

Admin commands (use admin token automatically):
  set-resource-policy    Set the KBS resource policy
  set-attestation-policy Set the attestation verification policy
  set-resource           Upload a confidential resource
  delete-resource        Delete a confidential resource
  get-reference-value    Get a reference value from RVPS
  set-sample-reference-value  Add a sample reference value to RVPS

Attestation commands (no admin token needed):
  get-resource           Get a confidential resource (requires attestation)
  attest                 Attestation and get attestation results token

Examples:
  $0 trusteeconfig-sample set-resource-policy --allow-all
  $0 trusteeconfig-sample set-resource-policy --deny-all
  $0 trusteeconfig-sample set-resource-policy --affirming
  $0 trusteeconfig-sample set-resource-policy --policy-file my-policy.rego
  $0 trusteeconfig-sample set-attestation-policy --policy-file policy.rego --type rego --id default
  $0 trusteeconfig-sample set-resource --path default/keys/my-key --resource-file key.bin
  $0 trusteeconfig-sample delete-resource --path default/keys/my-key
  $0 trusteeconfig-sample get-resource --path default/keys/my-key

Environment variables:
  NAMESPACE    Kubernetes namespace (default: default)
  LOCAL_PORT   Local port for port-forward (default: 8080)
  KBS_PORT     KBS service port (default: 8080)
EOF
    exit 0
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" || $# -eq 0 ]]; then
    usage
fi

NAMESPACE="${NAMESPACE:-default}"
KBS_PORT="${KBS_PORT:-8080}"
LOCAL_PORT="${LOCAL_PORT:-8080}"

ALL_COMMANDS="get-resource attest set-resource-policy set-attestation-policy set-resource delete-resource get-reference-value set-sample-reference-value"

# The release name is optional: when the first argument is already a known
# command (optionally preceded by "config"), auto-detect the release instead.
FIRST_ARG="${1:-}"
if [[ "$FIRST_ARG" == "config" ]]; then
    if [[ " $ALL_COMMANDS " == *" ${2:-} "* ]]; then
        RELEASE_NAME=""
    else
        RELEASE_NAME="${2:?Usage: $0 [release-name] <command> [args...]}"
        shift
    fi
elif [[ " $ALL_COMMANDS " == *" $FIRST_ARG "* ]]; then
    RELEASE_NAME=""
else
    RELEASE_NAME="$FIRST_ARG"
    shift
fi

if [[ -z "$RELEASE_NAME" ]]; then
    mapfile -t CANDIDATES < <(
        kubectl get secrets -n "$NAMESPACE" -o name 2>/dev/null \
            | sed 's#^secret/##' \
            | grep -- '-bootstrap-user-keys$' \
            | sed 's#-bootstrap-user-keys$##' \
            | sort -u
    )
    if [[ ${#CANDIDATES[@]} -eq 0 ]]; then
        echo "Error: could not auto-detect a release in namespace '$NAMESPACE'." >&2
        echo "Pass the release name explicitly: $0 <release-name> <command> [args...]" >&2
        exit 1
    elif [[ ${#CANDIDATES[@]} -gt 1 ]]; then
        echo "Error: multiple releases found in namespace '$NAMESPACE':" >&2
        printf '  %s\n' "${CANDIDATES[@]}" >&2
        echo "Pass the release name explicitly: $0 <release-name> <command> [args...]" >&2
        exit 1
    fi
    RELEASE_NAME="${CANDIDATES[0]}"
    echo "Using release: $RELEASE_NAME" >&2
fi

# Strip "config" if the user passed it — the script adds it automatically for admin commands
if [[ "${1:-}" == "config" ]]; then
    shift
fi

SECRET_NAME="${RELEASE_NAME}-bootstrap-user-keys"
SVC_NAME="${RELEASE_NAME}-kbs"

# Determine if this is a top-level command (no admin token / no config wrapper)
TOPLEVEL_COMMANDS="get-resource attest"
IS_TOPLEVEL=false
for cmd in $TOPLEVEL_COMMANDS; do
    if [[ "${1:-}" == "$cmd" ]]; then
        IS_TOPLEVEL=true
        break
    fi
done

if ! command -v kbs-client &>/dev/null; then
    echo "Error: kbs-client not found in PATH" >&2
    exit 1
fi

ADMIN_TOKEN=$(kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.data.KBS_ADMIN_TOKEN}' | base64 -d)

if [[ -z "$ADMIN_TOKEN" ]]; then
    echo "Error: could not extract KBS_ADMIN_TOKEN from secret $SECRET_NAME" >&2
    exit 1
fi

TOKEN_FILE=$(mktemp)
trap 'rm -f "$TOKEN_FILE"' EXIT
echo -n "$ADMIN_TOKEN" > "$TOKEN_FILE"

kubectl port-forward "svc/$SVC_NAME" "$LOCAL_PORT:$KBS_PORT" -n "$NAMESPACE" &
PF_PID=$!
trap 'kill $PF_PID 2>/dev/null; rm -f "$TOKEN_FILE"' EXIT

sleep 2

if [[ "$IS_TOPLEVEL" == true ]]; then
    kbs-client --url "http://localhost:$LOCAL_PORT" "$@"
else
    kbs-client --url "http://localhost:$LOCAL_PORT" config --admin-token-file "$TOKEN_FILE" "$@"
fi

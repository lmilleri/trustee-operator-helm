#!/bin/bash
#
# Decode a Confidential Containers / Trustee attestation token (EAT JWT).
#
# The script splits the compact JWT, base64url-decodes the header and payload,
# pretty-prints them, shows the signing key's X.509 chain and, optionally,
# verifies the ES256 signature against the embedded leaf certificate.
#
# Dependencies: bash, base64, openssl, jq (recommended), python3 (--verify only)

set -euo pipefail

usage() {
    cat <<EOF
Usage: $0 [options] [token|file]

Decode a Trustee attestation token. The token can be passed as an argument,
read from a file with -f, or piped on stdin.

Options:
  -f, --file FILE   Read the token from FILE ("-" reads stdin)
  -c, --certs       Show the x5c certificate chain details (default)
  -C, --no-certs    Do not show the certificate chain
  -V, --verify      Verify the ES256 signature with the leaf certificate
  -p, --payload     Print only the decoded payload
  -r, --raw         Do not pretty-print JSON (emit compact JSON)
  -h, --help        Show this help

Examples:
  $0 eyJ0eXAiOiJKV1Q...
  $0 -f token.txt
  cat token.txt | $0
  $0 -V -f token.txt
EOF
}

die() {
    echo "Error: $*" >&2
    exit 1
}

# Pick a portable base64 decoder (GNU uses -d, BSD/macOS uses -D).
if printf 'aGk=' | base64 -d >/dev/null 2>&1; then
    B64_DECODE=(base64 -d)
else
    B64_DECODE=(base64 -D)
fi

b64url_decode() {
    local data="$1"
    data="${data//-/+}"
    data="${data//_//}"
    case $(( ${#data} % 4 )) in
        2) data+="==" ;;
        3) data+="=" ;;
        1) die "invalid base64url input" ;;
    esac
    printf '%s' "$data" | "${B64_DECODE[@]}"
}

fmt_time() {
    local ts="$1"
    if date -u -d "@$ts" '+%Y-%m-%dT%H:%M:%SZ' >/dev/null 2>&1; then
        date -u -d "@$ts" '+%Y-%m-%dT%H:%M:%SZ'
    elif date -u -r "$ts" '+%Y-%m-%dT%H:%M:%SZ' >/dev/null 2>&1; then
        date -u -r "$ts" '+%Y-%m-%dT%H:%M:%SZ'
    else
        printf '%s' "$ts"
    fi
}

pretty() {
    if [[ "$RAW" == true ]]; then
        cat
    elif command -v jq >/dev/null 2>&1; then
        jq .
    elif command -v python3 >/dev/null 2>&1; then
        python3 -m json.tool
    else
        cat
    fi
}

SHOW_CERTS=true
VERIFY=false
PAYLOAD_ONLY=false
RAW=false
INPUT=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -f|--file)   INPUT="${2:?--file requires an argument}"; shift 2 ;;
        -c|--certs)  SHOW_CERTS=true; shift ;;
        -C|--no-certs) SHOW_CERTS=false; shift ;;
        -V|--verify) VERIFY=true; shift ;;
        -p|--payload) PAYLOAD_ONLY=true; shift ;;
        -r|--raw)    RAW=true; shift ;;
        -h|--help)   usage; exit 0 ;;
        --)          shift; break ;;
        -*)          die "unknown option: $1 (see --help)" ;;
        *)           INPUT="$1"; shift ;;
    esac
done

if [[ -z "$INPUT" ]]; then
    if [[ ! -t 0 ]]; then
        INPUT="$(cat)"
    else
        die "no token given (pass a token, use -f FILE, or pipe it on stdin)"
    fi
elif [[ "$INPUT" == "-" ]]; then
    INPUT="$(cat)"
elif [[ -f "$INPUT" ]]; then
    INPUT="$(cat "$INPUT")"
fi

TOKEN="$(printf '%s' "$INPUT" | tr -d '[:space:]')"
[[ -n "$TOKEN" ]] || die "empty token"

IFS='.' read -r HDR PAY SIG REST <<< "$TOKEN"
[[ -n "${HDR:-}" && -n "${PAY:-}" && -n "${SIG:-}" ]] \
    || die "not a compact JWT (expected 3 dot-separated parts)"
[[ -z "${REST:-}" ]] || die "unexpected extra segment in token"

HEADER_JSON="$(b64url_decode "$HDR")"
PAYLOAD_JSON="$(b64url_decode "$PAY")"

WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/decode-eat.XXXXXX")"
trap 'rm -rf "$WORKDIR"' EXIT

if [[ "$PAYLOAD_ONLY" == true ]]; then
    printf '%s' "$PAYLOAD_JSON" | pretty
    exit 0
fi

printf '=== Header ===\n'
printf '%s' "$HEADER_JSON" | pretty

printf '\n=== Payload ===\n'
printf '%s' "$PAYLOAD_JSON" | pretty

printf '\n=== Signature ===\n'
printf '%s\n' "$SIG"
printf 'length: %s bytes (base64url)\n' "$(printf '%s' "$SIG" | wc -c)"

# Extract the embedded x5c chain once; used for display and verification.
X5C=()
if command -v jq >/dev/null 2>&1; then
    mapfile -t X5C < <(
        printf '%s' "$HEADER_JSON" \
            | jq -r '(.jwk.x5c // .x5c // [])[]' 2>/dev/null || true
    )
fi

if [[ "$SHOW_CERTS" == true ]]; then
    printf '\n=== Certificates ===\n'
    if [[ ${#X5C[@]} -eq 0 ]]; then
        printf 'no embedded x5c chain found\n'
    else
        for i in "${!X5C[@]}"; do
            cert_der="$WORKDIR/cert$i.der"
            cert_pem="$WORKDIR/cert$i.pem"
            b64url_decode "${X5C[$i]}" > "$cert_der"
            openssl x509 -inform DER -in "$cert_der" -out "$cert_pem" 2>/dev/null \
                || { printf -- '--- Certificate [%s]: failed to parse ---\n' "$i"; continue; }

            printf -- '--- Certificate [%s] ---\n' "$i"
            openssl x509 -in "$cert_pem" -noout -subject -issuer -dates 2>/dev/null \
                | sed 's/^/  /'
            openssl x509 -in "$cert_pem" -noout -fingerprint -sha256 2>/dev/null \
                | sed 's/^/  /'
            openssl x509 -in "$cert_pem" -noout -ext subjectAltName 2>/dev/null \
                | sed 's/^/  /' || true
        done
    fi
fi

if command -v jq >/dev/null 2>&1; then
    ISS="$(printf '%s' "$PAYLOAD_JSON" | jq -r '.iss // "-"')"
    STATUS="$(printf '%s' "$PAYLOAD_JSON" | jq -r '.ear_status // "-"')"
    IAT="$(printf '%s' "$PAYLOAD_JSON" | jq -r '.iat // empty')"
    EXP="$(printf '%s' "$PAYLOAD_JSON" | jq -r '.exp // empty')"

    printf '\n=== Summary ===\n'
    printf 'issuer:     %s\n' "$ISS"
    printf 'ear_status: %s\n' "$STATUS"
    if [[ -n "$IAT" ]]; then
        printf 'issued:     %s (%s)\n' "$(fmt_time "$IAT")" "$IAT"
    fi
    if [[ -n "$EXP" ]]; then
        printf 'expires:    %s (%s)\n' "$(fmt_time "$EXP")" "$EXP"
    fi
    if [[ -n "$IAT" && -n "$EXP" ]]; then
        printf 'ttl:        %ss\n' "$((EXP - IAT))"
    fi
fi

if [[ "$VERIFY" == true ]]; then
    printf '\n=== Signature verification ===\n'
    command -v python3 >/dev/null 2>&1 || die "--verify requires python3"
    [[ ${#X5C[@]} -gt 0 ]] || die "--verify requires an embedded x5c chain"

    b64url_decode "${X5C[0]}" > "$WORKDIR/leaf.der"
    openssl x509 -inform DER -in "$WORKDIR/leaf.der" -pubkey -noout \
        > "$WORKDIR/pubkey.pem" 2>/dev/null \
        || die "could not extract public key from leaf certificate"

    b64url_decode "$SIG" > "$WORKDIR/sig.raw"
    SIGLEN="$(wc -c < "$WORKDIR/sig.raw")"
    [[ "$SIGLEN" -eq 64 ]] \
        || die "unexpected signature length: $SIGLEN bytes (ES256 expects 64)"

    # Convert the raw R||S ES256 signature to DER for openssl.
    python3 - "$WORKDIR/sig.raw" "$WORKDIR/sig.der" <<'PY'
import sys

raw = open(sys.argv[1], "rb").read()
half = len(raw) // 2


def der_int(value):
    data = value.to_bytes((value.bit_length() + 7) // 8 or 1, "big")
    if data[0] & 0x80:
        data = b"\x00" + data
    return b"\x02" + bytes([len(data)]) + data


body = der_int(int.from_bytes(raw[:half], "big")) + der_int(int.from_bytes(raw[half:], "big"))
open(sys.argv[2], "wb").write(b"\x30" + bytes([len(body)]) + body)
PY

    printf '%s.%s' "$HDR" "$PAY" > "$WORKDIR/signed.txt"
    if openssl dgst -sha256 -verify "$WORKDIR/pubkey.pem" \
            -signature "$WORKDIR/sig.der" "$WORKDIR/signed.txt" >/dev/null 2>&1; then
        printf 'result: VALID (ES256, leaf certificate)\n'
    else
        printf 'result: INVALID\n'
        exit 1
    fi
fi

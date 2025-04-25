#!/bin/bash

VALIDITY_DAYS=365
OUTPUT_DIR="./certs"

mkdir -p "$OUTPUT_DIR"
generate_cert_and_key() {
    local cert_name=$1
    local key_name=$2
    local common_name=$3

    openssl genrsa                          \
        -out "$OUTPUT_DIR/$key_name" 2048

    openssl req                             \
        -new                                \
        -key "$OUTPUT_DIR/$key_name"        \
        -out "$OUTPUT_DIR/$cert_name.csr"   \
        -subj "/CN=$common_name"

    openssl x509                            \
        -req                                \
        -days $VALIDITY_DAYS                \
        -in "$OUTPUT_DIR/$cert_name.csr"    \
        -signkey "$OUTPUT_DIR/$key_name"    \
        -out "$OUTPUT_DIR/$cert_name"

    rm "$OUTPUT_DIR/$cert_name.csr"
}

generate_cert_and_key           \
    "encrypt_cert.pem"          \
    "encrypt_key.pem"           \
        "Encryption Certificate"

generate_cert_and_key           \
    "sign_cert.pem"             \
    "sign_key.pem"              \
        "Signing Certificate"

echo "Certificates and keys have been generated in the '$OUTPUT_DIR' directory."

#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later

set -euo pipefail

home_dir="${HOME:-$(cd ~ && pwd)}"
output_dir="${1:-$home_dir/.choysum/cert}"
output_dir="${output_dir%/}"

mkdir -p "$output_dir"

ca_key="$output_dir/ca.key"
ca_csr="$output_dir/ca.csr"
ca_crt="$output_dir/ca.crt"
server_key="$output_dir/server.key"
server_csr="$output_dir/server.csr"
server_crt="$output_dir/server.crt"

san_config="$(mktemp "${TMPDIR:-/tmp}/choysum-local-tls.XXXXXX.cnf")"
cleanup() {
	rm -f "$san_config"
}
trap cleanup EXIT

cat >"$san_config" <<'EOF'
[req]
prompt = no
distinguished_name = req_distinguished_name

[req_distinguished_name]
C = CN
L = Choysum Town
O = Choysum
CN = Choysum Dev Server

[SAN]
subjectAltName = DNS:localhost,DNS:testhost,IP:127.0.0.1
EOF

openssl genrsa -out "$ca_key" 2048
openssl req -new -key "$ca_key" -out "$ca_csr" \
	-subj "/C=CN/L=Choysum Town/O=Choysum/CN=Choysum Dev CA"
openssl req -new -x509 -days 3650 -key "$ca_key" -out "$ca_crt" -sha256 \
	-subj "/C=CN/L=Choysum Town/O=Choysum/CN=Choysum Dev CA"

openssl genrsa -out "$server_key" 2048
openssl req -new -key "$server_key" -out "$server_csr" \
	-config "$san_config" \
	-reqexts SAN
openssl x509 -req -days 3650 \
	-in "$server_csr" -out "$server_crt" -sha256 \
	-CA "$ca_crt" -CAkey "$ca_key" -CAcreateserial \
	-extensions SAN \
	-extfile "$san_config"

chmod 600 "$ca_key" "$server_key"

cat <<EOF
Generated local TLS files in $output_dir

Use these paths for local TLS config:
  server.enabledTLS=true
  server.tlsCaFile=$ca_crt
  server.tlsCertFile=$server_crt
  server.tlsKeyFile=$server_key
EOF
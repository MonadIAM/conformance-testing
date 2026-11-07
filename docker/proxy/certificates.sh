#!/bin/sh
set -eu
umask 077

if [ -s /certs/ca.crt ] && [ -s /certs/server.crt ] && [ -s /certs/server.key ] &&
	openssl verify -CAfile /certs/ca.crt /certs/server.crt &&
	openssl x509 -checkend 86400 -noout -in /certs/server.crt &&
	openssl verify -CAfile /certs/ca.crt -verify_hostname localhost /certs/server.crt &&
	openssl verify -CAfile /certs/ca.crt -verify_hostname conformance.test /certs/server.crt &&
	openssl verify -CAfile /certs/ca.crt -verify_hostname iam.test /certs/server.crt; then
	exit 0
fi

mkdir -p /certs

if [ ! -s /certs/ca.crt ] || [ ! -s /certs/ca.key ]; then
	openssl req -x509 \
		-newkey rsa:3072 \
		-nodes \
		-sha256 \
		-days 3650 \
		-keyout /certs/ca.key \
		-out /certs/ca.crt \
		-subj '/CN=MonadIAM Local Conformance CA' \
		-addext 'basicConstraints=critical,CA:TRUE' \
		-addext 'keyUsage=critical,keyCertSign,cRLSign'
fi

openssl req -new \
	-newkey rsa:2048 \
	-nodes \
	-sha256 \
	-keyout /certs/server.key \
	-out /certs/server.csr \
	-subj '/CN=conformance.test'

cat > /certs/server.ext <<-'EOF'
	basicConstraints=critical,CA:FALSE
	keyUsage=critical,digitalSignature,keyEncipherment
	extendedKeyUsage=serverAuth
	subjectAltName=DNS:conformance.test,DNS:iam.test,DNS:localhost
	EOF

openssl x509 -req \
	-in /certs/server.csr \
	-CA /certs/ca.crt \
	-CAkey /certs/ca.key \
	-CAcreateserial \
	-out /certs/server.crt \
	-days 365 \
	-sha256 \
	-extfile /certs/server.ext

chmod 644 /certs/ca.crt /certs/server.crt
rm /certs/server.csr /certs/server.ext

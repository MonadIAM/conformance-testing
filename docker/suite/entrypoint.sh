#!/bin/sh
set -eu

cp "$JAVA_HOME/lib/security/cacerts" /tmp/conformance-cacerts
keytool -importcert -noprompt -alias monadiam-local-conformance \
	-file /certs/ca.crt -keystore /tmp/conformance-cacerts -storepass changeit

exec java \
	-Djavax.net.ssl.trustStore=/tmp/conformance-cacerts \
	-Djavax.net.ssl.trustStorePassword=changeit \
	-Djdk.tls.maxHandshakeMessageSize=65536 \
	-Dfintechlabs.base_url=https://conformance.test:8443 \
	-Dspring.mongodb.uri=mongodb://mongodb:27017/test_suite \
	-Doidc.google.clientid=local-unused \
	-Doidc.google.secret=local-unused \
	-Doidc.gitlab.clientid=local-unused \
	-Doidc.gitlab.secret=local-unused \
	-jar /server/fapi-test-suite.jar

.PHONY: check fmt build pull up down stop logs status smoke check-iam test-config conformance

export GOCACHE := $(CURDIR)/.local/go-build
PLAN ?= oidcc-config-certification-test-plan
CONFIG ?= configs/local.example.json
ARGS ?=

check:
	docker compose config --quiet
	sh -n docker/proxy/certificates.sh
	sh -n docker/suite/entrypoint.sh
	@test -z "$$(gofmt -l cmd)" || { gofmt -l cmd; exit 1; }
	go vet ./...

fmt:
	gofmt -w cmd

build:
	go build -o bin/conformance ./cmd

pull:
	docker compose pull

up: check
	mkdir -p .local/certs
	@if [ -s .local/certs/ca.crt ]; then \
		docker compose up -d; \
	else \
		docker compose up -d --force-recreate; \
	fi

down:
	docker compose down

stop:
	docker compose stop

logs:
	docker compose logs --tail=100 --follow server proxy

status:
	docker compose ps -a

smoke:
	curl --fail --silent --show-error --max-time 15 --cacert .local/certs/ca.crt \
	  --resolve conformance.test:8443:127.0.0.1 \
	  https://conformance.test:8443/ > /dev/null

check-iam:
	docker compose exec -T server curl --fail --silent --show-error --max-time 15 \
	  --cacert /certs/ca.crt https://localhost:4002/.well-known/openid-configuration

test: build
	./bin/conformance -plan oidcc-config-certification-test-plan -config "$(CONFIG)"

conformance: build
	./bin/conformance -plan "$(PLAN)" -config "$(CONFIG)" $(ARGS)

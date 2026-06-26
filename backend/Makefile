-include .env
export

.PHONY: run build docker-up docker-down migrate-up migrate-down docker-logs test test-unit test-integration test-all cover check-coverage open-integration-sum

run:
	go run ./cmd/marshal

build:
	go build -o bin/marshal ./cmd/marshal

docker-up:
	docker-compose -f ../docker-compose.yaml up -d

docker-down:
	docker-compose -f ../docker-compose.yaml down

docker-logs:
	docker-compose -f ../docker-compose.yaml logs -f

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

# Run all tests with race detection
test:
	go test -v -race ./...

# Run only pure unit tests (no DB needed)
test-unit:
	go test -v -race ./internal/math/... ./internal/handlers/... ./internal/grouper/... ./internal/assigner/... ./internal/worker/... ./internal/ws/...

# Run integration tests (requires Postgres)
test-integration:
	@set -o pipefail; \
	SECONDS=0; \
	go test -v -race -count=1 -timeout 180s -tags=integration \
		./internal/store/... \
		> /tmp/marshal_int.log 2>&1; \
	STATUS=$$?; \
	ELAPSED=$$SECONDS; \
	PASSED=$$(grep -cE "^(    )?--- PASS:" /tmp/marshal_int.log || true); \
	FAILED=$$(grep -cE "^(    )?--- FAIL:" /tmp/marshal_int.log || true); \
	{ \
		printf "\n"; \
		printf "  ┌─────────────────────────────────────────────────────────────────────────────────────┬────────┬────────┐\n"; \
		printf "  │ %-83s │ %-6s │ %-6s │\n", "TEST NAME", "STATUS", "TIME"; \
		printf "  ├─────────────────────────────────────────────────────────────────────────────────────┼────────┼────────┤\n"; \
		grep -E "^(    )?--- (PASS|FAIL):" /tmp/marshal_int.log \
			| sed 's/^[[:space:]]*//' \
			| awk '{ \
				status=$$2; sub(":","",status); \
				name=$$3; \
				time=$$4; gsub(/[()s]/,"",time); \
				if (time == "") time="-"; \
				printf "  │ %-83s │ %-6s │ %-5ss │\n", name, status, time; \
			}'; \
		printf "  └─────────────────────────────────────────────────────────────────────────────────────┴────────┴────────┘\n"; \
		printf "\n"; \
	} >> /tmp/marshal_int.log; \
	printf "\n"; \
	printf "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"; \
	printf "      Marshal Integration Test Summary\n"; \
	printf "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"; \
	printf "      Passed : %s\n" "$$PASSED"; \
	printf "      Failed : %s\n" "$$FAILED"; \
	printf "      Time   : %ss / 180s\n" "$$ELAPSED"; \
	if [ "$$FAILED" -gt 0 ]; then \
		printf "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"; \
		printf "      Failures\n"; \
		printf "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"; \
		grep -E "^(    )?--- FAIL:" /tmp/marshal_int.log \
			| sed 's/^[[:space:]]*//' \
			| awk '{ name=$$3; printf "      ✗  %s\n", name }'; \
		printf "\n"; \
		grep -B1 -A3 "FAIL:" /tmp/marshal_int.log \
			| grep -E "\.go:[0-9]+" \
			| sed 's/^[[:space:]]*//' \
			| awk '{ printf "         %s\n", $$0 }'; \
	fi; \
	printf "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"; \
	printf "\n"; \
	exit $$STATUS

# Open the integration test log file
open-integration-sum:
	open /tmp/marshal_int.log || code /tmp/marshal_int.log

# Run ALL tests (unit + integration)
test-all:
	go test -v -race -count=1 ./internal/...

# Generate HTML coverage report
cover:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | grep "^total:"

# Check filtered coverage (excludes boilerplate). Threshold: 40%.
COVERAGE_THRESHOLD := 40
check-coverage:
	@go test -coverprofile=coverage.out -tags=integration ./internal/... || { echo "  x  FAIL: Tests failed to build or execute. Coverage check aborted."; exit 1; }; \
	grep -v -E "github.com/Ksalgotra1/Marshal/(cmd|internal/api|internal/config)" coverage.out > coverage_filtered.out; \
	COVERAGE=$$(go tool cover -func=coverage_filtered.out | grep "^total:" | awk '{print $$NF}' | tr -d '%'); \
	printf "\n  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"; \
	printf "      Coverage Report\n"; \
	printf "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"; \
	printf "      Filtered Total : %s%%\n" "$$COVERAGE"; \
	printf "      Threshold      : $(COVERAGE_THRESHOLD)%%\n"; \
	printf "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"; \
	if [ $$(echo "$${COVERAGE} < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "  x  FAIL: Coverage $${COVERAGE}% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	else \
		echo "  o  PASS: Coverage $${COVERAGE}% meets threshold $(COVERAGE_THRESHOLD)%"; \
	fi

.DEFAULT_GOAL := run

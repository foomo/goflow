.DEFAULT_GOAL:=help
-include .makerc

# --- Config -----------------------------------------------------------------

# Newline hack for error output
define br


endef

# --- Targets -----------------------------------------------------------------

# This allows us to accept extra arguments
%: .mise .lefthook
	@:

.PHONY: .mise
# Install dependencies
.mise:
ifeq (, $(shell command -v mise))
	$(error $(br)$(br)Please ensure you have 'mise' installed and activated!$(br)$(br)  $$ brew update$(br)  $$ brew install mise$(br)$(br)See the documentation: https://mise.jdx.dev/getting-started.html)
endif
	@mise install

.PHONY: .lefthook
# Configure git hooks for lefthook
.lefthook:
	@lefthook install --reset-hooks-path

### Tasks

.PHONY: check
## Run lint & test
check: tidy generate lint test audit

.PHONY: tidy
## Run go mod tidy
tidy:
	@echo "〉go mod tidy"
	@go mod tidy

.PHONY: lint
## Run linter
lint:
	@echo "〉golangci-lint run"
	@golangci-lint run

.PHONY: lint.fix
## Fix lint violations
lint.fix:
	@echo "〉golangci-lint run fix"
	@golangci-lint run --fix

.PHONY: lint.branch
## Run linter with --new-from-rev=origin/main
lint.branch:
	@echo "〉golangci-lint run with --new-from-rev=origin/main"
	@golangci-lint run --new-from-rev=origin/main

.PHONY: test
## Run tests
test:
	@echo "〉go test"
	@GO_TEST_TAGS=-skip go test -coverprofile=coverage.out -tags=safe ./...

.PHONY: test.race
## Run tests with -race
test.race:
	@echo "〉go test -race"
	@GO_TEST_TAGS=-skip go test -coverprofile=coverage.out -tags=safe -race ./...

.PHONY: test.nocache
## Run tests with -count=1
test.nocache:
	@echo "〉go test -count=1"
	@GO_TEST_TAGS=-skip go test -coverprofile=coverage.out -tags=safe -count=1 ./...

.PHONY: test.bench
## Run tests with -bench
test.bench:
	@echo "〉go test -bench"
	@GO_TEST_TAGS=-skip go test -tags=safe -bench=. -benchmem -count=10 ./... > .benchmark.txt | benchstat benchmark.txt .benchmark.txt
	@rm .benchstat.txt

.PHONY: test.bench.update
## Run tests with -bench & update baseline.txt
test.bench.update:
	@echo "〉go test -bench (updating baseline)"
	@GO_TEST_TAGS=-skip go test -tags=safe -bench=. -benchmem -count=10 ./... > benchmark.txt
	@echo "✅ benchmark.txt updated"

.PHONY: generate
## Run go generate
generate:
	@echo "〉go generate"
	@go generate ./...

### Dependencies

.PHONY: audit
## Run security audit
audit:
	@echo "〉security audit"
	#@trivy fs . --format table --severity HIGH,CRITICAL
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go govulncheck ./...

.PHONY: outdated
## Show outdated direct dependencies
outdated:
	@echo "〉mise"
	@mise outdated -l --local
	@echo "〉go mod outdated"
	@go list -u -m -json all | go-mod-outdated -update -direct

.PHONY: upgrade
## Show outdated direct dependencies
upgrade: go.work
	@echo "〉go mod upgrade"
	@go get -u ./...

### Documentation

.PHONY: docs
## Open docs
docs:
	@echo "〉starting docs"
	@cd docs && bun install && bun run dev

.PHONY: docs.build
## Open docs
docs.build:
	@echo "〉building docs"
	@cd docs && bun install && bun run build

.PHONY: godocs
## Open go docs
godocs:
	@echo "〉starting go docs"
	@go doc -http

### Utils

.PHONY: help
## Show help text
help:
	@echo ""
	@echo "goflow"
	@echo ""
	@echo "Usage:\n  make [task]"
	@awk '{ \
		if($$0 ~ /^### /){ \
			if(help) printf "%-23s %s\n\n", cmd, help; help=""; \
			printf "\n%s:\n", substr($$0,5); \
		} else if($$0 ~ /^[a-zA-Z0-9._-]+:/){ \
			cmd = substr($$0, 1, index($$0, ":")-1); \
			if(help) printf "  %-23s %s\n", cmd, help; help=""; \
		} else if($$0 ~ /^##/){ \
			help = help ? help "\n                        " substr($$0,3) : substr($$0,3); \
		} else if(help){ \
			print "\n                        " help "\n"; help=""; \
		} \
	}' $(MAKEFILE_LIST)

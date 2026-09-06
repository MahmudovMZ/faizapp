.PHONY: fmt fmt-check mod-verify vet test build secret-check whitespace-check check vuln hooks

fmt:
	gofmt -w .

fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		echo "The following files are not gofmt-formatted:"; \
		echo "$$files"; \
		exit 1; \
	fi

mod-verify:
	go mod verify

vet:
	go vet ./...

test:
	go test -race -cover ./...

build:
	go build ./...

secret-check:
	@if git ls-files --error-unmatch .env >/dev/null 2>&1; then \
		echo ".env is tracked by Git. Remove it before pushing."; \
		exit 1; \
	fi

whitespace-check:
	git diff --check

check: fmt-check whitespace-check mod-verify secret-check vet test build
	@echo "All local checks passed."

vuln:
	@command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-push
	@echo "Git hooks enabled."


.PHONY: fmt test race vet check verify docs docs-check omemo-example zip

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

check: fmt test vet docs-check

verify: check race omemo-example

docs:
	./scripts/generate-docs.sh

docs-check:
	@tmp=$$(mktemp -d); \
	trap 'rm -rf $$tmp' EXIT; \
	./scripts/generate-docs.sh $$tmp; \
	cmp -s $$tmp/xep-support.md docs/xep-support.md || { echo 'docs/xep-support.md is stale; run make docs' >&2; exit 1; }; \
	diff -ru docs/api $$tmp/api >/dev/null || { echo 'docs/api is stale; run make docs' >&2; exit 1; }

omemo-example:
	@go run ./examples/omemo >/dev/null

zip: verify docs
	mkdir -p dist
	zip -qr dist/slixmpp-go.zip . -x 'dist/*' '.git/*'

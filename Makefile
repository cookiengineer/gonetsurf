# gonetsurf build and compliance targets.
#
# `make compliance-tests` is the single entry point used to prove the
# implementation against the pinned specifications and vendored corpora.

GO ?= go
FUZZTIME ?= 10s
BENCHTIME ?= 100x
FORMAT_DIRS := cmd urls parsers protocols caches storages internal dom styles layouts

.PHONY: all build vet fmt fmt-check deps-check test conformance compliance-tests fuzz bench corpus-refresh spec-refresh clean

all: compliance-tests

build:
	$(GO) build ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w $(FORMAT_DIRS)

fmt-check:
	@unformatted="$$(gofmt -l $(FORMAT_DIRS))"; \
	if [ -n "$$unformatted" ]; then echo "unformatted files:"; echo "$$unformatted"; exit 1; fi

deps-check:
	$(GO) run ./cmd/netsurfctl deps-check

# All unit, conformance, and E2E tests.
test:
	$(GO) test ./... -count=1

# Only the specification conformance suites.
conformance:
	$(GO) test ./... -run Conformance -count=1

# The single compliance entry point: static checks plus all tests.
compliance-tests: build vet fmt-check deps-check test

# Short fuzz runs for the parser packages (override FUZZTIME for longer runs).
fuzz:
	$(GO) test ./urls/ -run=^$$ -fuzz=FuzzParse -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/tokenizer/ -run=^$$ -fuzz=FuzzTokenize -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/parser/ -run=^$$ -fuzz=FuzzParseStylesheet -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/grammar/ -run=^$$ -fuzz=FuzzMatchDefinition -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/selectors/ -run=^$$ -fuzz=FuzzParse -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/values/ -run=^$$ -fuzz=FuzzMath -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/cssom/ -run=^$$ -fuzz=FuzzParseSerialize -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/properties/ -run=^$$ -fuzz=FuzzValidate -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/ -run=^$$ -fuzz=FuzzCSS -fuzztime=$(FUZZTIME)
	$(GO) test ./dom/ -run=^$$ -fuzz=FuzzSelectors -fuzztime=$(FUZZTIME)
	$(GO) test ./protocols/http1/ -run=^$$ -fuzz=FuzzReadResponse -fuzztime=$(FUZZTIME)

bench:
	$(GO) test ./parsers/css/ -run=^$$ -bench=. -benchtime=$(BENCHTIME)
	$(GO) test ./layouts/ -run=^$$ -bench=. -benchtime=$(BENCHTIME)

# Re-fetch raw upstream test sources into testdata/upstream and regenerate the
# normalized corpora consumed by the tests.
corpus-refresh:
	$(GO) run ./cmd/corpusgen -refresh

# Re-fetch the pinned specification snapshots.
spec-refresh:
	REFRESH=1 sh specifications/update.sh

clean:
	$(GO) clean ./...

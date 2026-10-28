# gonetsurf build and compliance targets.
#
# `make compliance-tests` is the single entry point used to prove the
# implementation against the pinned specifications and vendored corpora.

GO ?= go
FUZZTIME ?= 10s
BENCHTIME ?= 100x
FORMAT_DIRS := cmd urls parsers protocols caches storages internal dom styles layouts rendering windowing

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
	$(GO) test ./urls/ -run=^$$ -fuzz=^FuzzParse$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/tokenizer/ -run=^$$ -fuzz=^FuzzTokenize$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/parser/ -run=^$$ -fuzz=^FuzzParseStylesheet$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/grammar/ -run=^$$ -fuzz=^FuzzMatchDefinition$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/selectors/ -run=^$$ -fuzz=^FuzzParse$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/values/ -run=^$$ -fuzz=^FuzzMath$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/cssom/ -run=^$$ -fuzz=^FuzzParseSerialize$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/properties/ -run=^$$ -fuzz=^FuzzValidate$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/ -run=^$$ -fuzz=^FuzzCSS$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./dom/ -run=^$$ -fuzz=^FuzzSelectors$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/html/ -run=^$$ -fuzz=^FuzzParse$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/html/ -run=^$$ -fuzz=^FuzzParseFragment$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/html/encodings/ -run=^$$ -fuzz=^FuzzSniff$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/html/microsyntaxes/ -run=^$$ -fuzz=^FuzzParsers$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./protocols/mime/ -run=^$$ -fuzz=^FuzzContentDisposition$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./protocols/dns/ -run=^$$ -fuzz=^FuzzParseMessage$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./protocols/websockets/ -run=^$$ -fuzz=^FuzzReadFrame$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./protocols/websockets/ -run=^$$ -fuzz=^FuzzClosePayload$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./protocols/http1/ -run=^$$ -fuzz=^FuzzReadResponse$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./caches/disk/ -run=^$$ -fuzz=^FuzzPathMapping$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./rendering/ -run=^$$ -fuzz=^FuzzRaster$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./rendering/ -run=^$$ -fuzz=^FuzzGradient$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./rendering/ -run=^$$ -fuzz=^FuzzTransformFunction$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./rendering/ -run=^$$ -fuzz=^FuzzSVG$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./rendering/images/ -run=^$$ -fuzz=^FuzzDecode$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./styles/ -run=^$$ -fuzz=^FuzzExpandDeclaration$$ -fuzztime=$(FUZZTIME)
	$(GO) test ./parsers/css/colors/ -run=^$$ -fuzz=^FuzzToSRGB$$ -fuzztime=$(FUZZTIME)

bench:
	$(GO) test ./parsers/css/ -run=^$$ -bench=. -benchtime=$(BENCHTIME)
	$(GO) test ./layouts/ -run=^$$ -bench=. -benchtime=$(BENCHTIME)
	$(GO) test ./rendering/ -run=^$$ -bench=. -benchtime=$(BENCHTIME)
	$(GO) test ./rendering/fonts/ -run=^$$ -bench=. -benchtime=$(BENCHTIME)

# Re-fetch raw upstream test sources into testdata/upstream and regenerate the
# normalized corpora consumed by the tests.
corpus-refresh:
	$(GO) run ./cmd/corpusgen -refresh

# Re-fetch the pinned specification snapshots.
spec-refresh:
	REFRESH=1 sh specifications/update.sh

clean:
	$(GO) clean ./...

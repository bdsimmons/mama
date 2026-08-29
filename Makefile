BIN := mama
PREFIX ?= $(HOME)/.local

build:
	go build -o $(BIN) .

test:
	go test ./...

vet:
	go vet ./...

install: build
	@mkdir -p $(PREFIX)/bin
	@install -m 0755 $(BIN) $(PREFIX)/bin/$(BIN)
	@mkdir -p $(PREFIX)/share/bash-completion/completions
	@$(PREFIX)/bin/$(BIN) completion bash > $(PREFIX)/share/bash-completion/completions/$(BIN) 2>/dev/null || true
	@echo "→ $(PREFIX)/bin/$(BIN)"

uninstall:
	rm -f $(PREFIX)/bin/$(BIN) $(PREFIX)/share/bash-completion/completions/$(BIN)

clean:
	rm -f $(BIN)

.PHONY: build test vet install uninstall clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Cross-compiled binaries for every platform Go targets by default. Static,
# no cgo, so each file is the whole install.
dist:
	@rm -rf dist && mkdir -p dist
	@for p in $(PLATFORMS); do \
	  os=$${p%/*}; arch=$${p#*/}; \
	  ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
	  out="dist/mama_$${os}_$${arch}$$ext"; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
	    go build -trimpath -ldflags "-s -w" -o $$out . || exit 1; \
	  echo "  $$out"; \
	done
	@cd dist && sha256sum * > SHA256SUMS && echo "  dist/SHA256SUMS"

.PHONY: dist

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

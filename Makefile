REPO_DIR := $(shell pwd)

.PHONY: build install symlink-agents symlink-skills symlink-statusline clean format check test

build:
	go build -o bip ./cmd/bip

install: symlink-agents symlink-skills symlink-statusline
	go install ./cmd/bip
	@echo "Installed bip (to \$$GOBIN if set, otherwise \$$HOME/go/bin)"
	@echo "Ensure the Go bin directory is in your PATH."

symlink-agents:
	mkdir -p ~/.claude/agents
	@for f in $(REPO_DIR)/agents/*.md; do \
		ln -sf "$$f" ~/.claude/agents/$$(basename "$$f"); \
	done
	@echo "Symlinked agents to ~/.claude/agents/"

symlink-skills:
	mkdir -p ~/.claude/skills
	@for d in $(REPO_DIR)/skills/*/; do \
		[ -d "$$d" ] && rm -rf ~/.claude/skills/$$(basename "$$d") && ln -s "$$d" ~/.claude/skills/$$(basename "$$d"); \
	done
	@echo "Symlinked skills to ~/.claude/skills/"

symlink-statusline:
	mkdir -p ~/.claude/statusline
	ln -sf $(REPO_DIR)/statusline/ctx_monitor.js ~/.claude/statusline/ctx_monitor.js
	@echo "Symlinked statusline to ~/.claude/statusline/"

clean:
	rm -f bip

# Code quality targets
format:
	go fmt ./...

check:
	go vet ./...

test:
	go test ./...

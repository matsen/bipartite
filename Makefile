REPO_DIR := $(shell pwd)

.PHONY: build install symlink-agents symlink-skills clean format check test

build:
	go build -o bip ./cmd/bip

install: symlink-agents symlink-skills
	go install ./cmd/bip
	@echo "Installed bip to \$$HOME/go/bin/bip"
	@echo "Ensure \$$HOME/go/bin is in your PATH (add to ~/.bashrc or ~/.zshrc):"
	@echo "  export PATH=\"\$$HOME/go/bin:\$$PATH\""

symlink-agents:
	mkdir -p ~/.claude/agents
	@for f in $(REPO_DIR)/agents/*.md; do \
		ln -sf "$$f" ~/.claude/agents/$$(basename "$$f"); \
	done
	@echo "Symlinked agents to ~/.claude/agents/"

symlink-skills:
	mkdir -p ~/.claude/skills
	@for d in $(REPO_DIR)/skills/*/; do \
		[ -d "$$d" ] && rm -f ~/.claude/skills/$$(basename "$$d") && ln -s "$$d" ~/.claude/skills/$$(basename "$$d"); \
	done
	@echo "Symlinked skills to ~/.claude/skills/"

clean:
	rm -f bip

# Code quality targets
format:
	go fmt ./...

check:
	go vet ./...

test:
	go test ./...

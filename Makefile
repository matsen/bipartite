REPO_DIR := $(shell pwd)
BIN_DIR := ~/bin

.PHONY: build install symlink-agents symlink-skills clean format check test

build:
	go build -o bip ./cmd/bip

install: build symlink-agents symlink-skills
	mkdir -p $(BIN_DIR)
	cp bip $(BIN_DIR)/bip
	@echo "Installed bip to $(BIN_DIR)/bip"
	@echo "Make sure $(BIN_DIR) is in your PATH."

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

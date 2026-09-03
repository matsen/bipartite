REPO_DIR := $(shell pwd)

.PHONY: build install symlink-agents symlink-skills symlink-statusline symlink-hooks clean format check test

build:
	go build -o bip ./cmd/bip

install: symlink-agents symlink-skills symlink-statusline symlink-hooks
	go install ./cmd/bip
	@echo "Installed bip (to \$$GOBIN if set, otherwise \$$HOME/go/bin)"
	@echo "Ensure the Go bin directory is in your PATH."

symlink-agents:
	mkdir -p ~/.claude/agents
	@for f in $(REPO_DIR)/agents/*.md; do \
		ln -sf "$$f" ~/.claude/agents/$$(basename "$$f"); \
	done
	@for l in ~/.claude/agents/*; do \
		if [ -L "$$l" ] && [ ! -e "$$l" ]; then \
			echo "Pruning stale symlink: $$l"; \
			rm -f "$$l"; \
		fi; \
	done
	@echo "Symlinked agents to ~/.claude/agents/"

symlink-skills:
	mkdir -p ~/.claude/skills
	@for d in $(REPO_DIR)/skills/*/; do \
		if [ -f "$$d/SKILL.md" ]; then \
			rm -rf ~/.claude/skills/$$(basename "$$d") && ln -s "$$d" ~/.claude/skills/$$(basename "$$d"); \
		fi; \
	done
	@rm -rf ~/.claude/skills/lib && ln -s $(REPO_DIR)/skills/lib ~/.claude/skills/lib
	@for l in ~/.claude/skills/*; do \
		if [ -L "$$l" ] && [ ! -e "$$l" ]; then \
			echo "Pruning stale symlink: $$l"; \
			rm -f "$$l"; \
		fi; \
	done
	@echo "Symlinked skills to ~/.claude/skills/"

symlink-hooks:
	mkdir -p ~/.claude/hooks
	@for f in $(REPO_DIR)/hooks/*.sh $(REPO_DIR)/hooks/*.py $(REPO_DIR)/hooks/termcheck-stamp; do \
		ln -sf "$$f" ~/.claude/hooks/$$(basename "$$f"); \
	done
	@echo "Symlinked hooks to ~/.claude/hooks/ (add them to settings.json -- see hooks/README.md)"

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

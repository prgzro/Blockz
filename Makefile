# ═══════════════════════════════════════════════════════════════
#  Blockz — Build System
#  A sovereign Layer-1 blockchain from scratch in Go.
# ═══════════════════════════════════════════════════════════════

BINARY   := Blockz
BIN_DIR  := ./bin
BUILD    := $(BIN_DIR)/$(BINARY)
GO       := go
GOFLAGS  := -v
LDFLAGS  := -s -w

.PHONY: build run dev clean test coverage lint fmt vet help

# ── Default target ────────────────────────────────────────────
all: build

# ── Build ─────────────────────────────────────────────────────

## build: Compile the Blockz binary
build:
	@echo "⛏️  Building $(BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD)
	@echo "✅ Binary ready → $(BUILD)"

# ── Run ───────────────────────────────────────────────────────

## run: Build and launch a single node
run: build
	$(BUILD)

## dev: Launch a 4-node local dev network with pre-funded accounts
dev: build
	$(BUILD) --dev --difficulty 8 --blocktime 3s

# ── Testing ───────────────────────────────────────────────────

## test: Run all tests (or filter by name: `make test TestVM`)
test:
	@$(GO) clean -testcache
	@if [ -z "$(filter-out test,$(MAKECMDGOALS))" ]; then \
		echo "🧪 Running all tests..."; \
		$(GO) test ./... -v -count=1; \
	else \
		name=$$(echo $(filter-out test,$(MAKECMDGOALS)) | sed 's/ /|/g'); \
		echo "🧪 Running test(s): $$name"; \
		$(GO) test ./... -v -count=1 -run "$$name"; \
	fi

## coverage: Run tests with coverage report
coverage:
	@echo "📊 Generating coverage report..."
	$(GO) test ./... -cover -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out
	@rm -f coverage.out

# ── Code Quality ──────────────────────────────────────────────

## fmt: Format all Go source files
fmt:
	$(GO) fmt ./...

## vet: Run Go vet for static analysis
vet:
	$(GO) vet ./...

## lint: Run fmt + vet
lint: fmt vet

# ── Maintenance ───────────────────────────────────────────────

## clean: Remove build artifacts and node data
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out
	@echo "✅ Clean."

## nuke: Clean everything including local chain data and logs
nuke: clean
	@rm -rf *_data/ blockz_data/ *.log
	@echo "💀 All node data destroyed."

# ── Help ──────────────────────────────────────────────────────

## help: Show this help
help:
	@echo ""
	@echo "⛓️  Blockz — available targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | sort
	@echo ""

# Catch-all for test name arguments
%::
	@:
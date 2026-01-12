.PHONY: build test clean install deps lint format run-example

# Variáveis
BINARY_NAME=migrator
BUILD_DIR=build
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-X main.appVersion=$(VERSION) -s -w

# Comandos principais
build: deps
	@echo "🚀 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/migrator
	@echo "✅ Build completed! Binary available at: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: deps
	@echo "📦 Building for multiple platforms..."
	@./scripts/build.sh

test:
	@echo "🧪 Running tests..."
	@./scripts/test.sh

deps:
	@echo "📥 Installing dependencies..."
	@go mod download
	@go mod tidy

lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint não encontrado. Executando go vet..."; \
		go vet ./...; \
	fi

format:
	@echo "🎨 Formatting code..."
	@go fmt ./...

clean:
	@echo "🧹 Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f *.log

install: build
	@echo "📦 Installing $(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ $(BINARY_NAME) installed to /usr/local/bin/"

uninstall:
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ $(BINARY_NAME) uninstalled"

# Exemplos de uso
run-example:
	@echo "📋 Listando workspaces..."
	@$(BUILD_DIR)/$(BINARY_NAME) list

dry-run:
	@echo "🧪 Executando dry-run da migração..."
	@$(BUILD_DIR)/$(BINARY_NAME) migrate --dry-run

migrate-batch:
	@echo "🚀 Executando migração com batch de 3..."
	@$(BUILD_DIR)/$(BINARY_NAME) migrate --batch-size 3

migrate-specific:
	@echo "🎯 Migrando projetos específicos..."
	@$(BUILD_DIR)/$(BINARY_NAME) migrate --projects "workspace1,workspace2"

# Comandos de desenvolvimento
dev: build
	@echo "🛠️  Executando em modo desenvolvimento..."
	@$(BUILD_DIR)/$(BINARY_NAME)

watch:
	@echo "👀 Observando mudanças..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "⚠️  'air' não encontrado. Instale com: go install github.com/cosmtrek/air@latest"; \
	fi

help:
	@echo "🔧 Comandos disponíveis:"
	@echo "  build         - Compila o binário"
	@echo "  build-all     - Compila para múltiplas plataformas"
	@echo "  test          - Executa testes"
	@echo "  deps          - Instala dependências"
	@echo "  lint          - Executa linter"
	@echo "  format        - Formata código"
	@echo "  clean         - Limpa arquivos temporários"
	@echo "  install       - Instala o binário"
	@echo "  uninstall     - Remove o binário"
	@echo ""
	@echo "📋 Comandos de exemplo:"
	@echo "  run-example   - Lista workspaces"
	@echo "  dry-run       - Executa migração simulada"
	@echo "  migrate-batch - Migração com batch customizado"
	@echo "  migrate-specific - Migra projetos específicos"
	@echo ""
	@echo "🛠️  Comandos de desenvolvimento:"
	@echo "  dev           - Executa em modo desenvolvimento"
	@echo "  watch         - Observa mudanças e recompila"
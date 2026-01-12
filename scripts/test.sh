#!/bin/bash

set -e

echo "🧪 Running tests for Terraform Cloud to S3 Migrator..."

# Executar testes unitários
echo "📋 Running unit tests..."
go test -v -race -coverprofile=coverage.out ./...

# Mostrar cobertura
echo "📊 Test coverage:"
go tool cover -func=coverage.out

# Gerar relatório HTML de cobertura (opcional)
if command -v open >/dev/null 2>&1; then
    echo "🌐 Generating HTML coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    echo "📱 Coverage report available at: coverage.html"
fi

echo "✅ Tests completed successfully!"
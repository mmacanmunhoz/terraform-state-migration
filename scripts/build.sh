#!/bin/bash

set -e

echo "🚀 Building Terraform Cloud to S3 Migrator..."

# Definir variáveis
BINARY_NAME="migrator"
BUILD_DIR="build"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
LDFLAGS="-X main.version=${VERSION} -s -w"

# Criar diretório de build
mkdir -p ${BUILD_DIR}

# Build para diferentes plataformas
echo "📦 Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ./cmd/migrator

echo "📦 Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ./cmd/migrator

echo "📦 Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ./cmd/migrator

echo "📦 Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ./cmd/migrator

# Build para plataforma atual
echo "📦 Building for current platform..."
go build -ldflags="${LDFLAGS}" -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/migrator

echo "✅ Build completed successfully!"
echo "📁 Binaries available in: ${BUILD_DIR}/"
ls -la ${BUILD_DIR}/
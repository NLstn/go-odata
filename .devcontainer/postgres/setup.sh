#!/bin/bash
set -e

echo "Setting up development environment..."

# Download Go modules
echo "📦 Downloading Go modules..."
go mod download

# Install golangci-lint
echo "🔍 Installing golangci-lint..."
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.5.0

# Install bombardier for load testing
echo "⚡ Installing bombardier..."
go install github.com/codesenberg/bombardier@latest

echo "✅ Development environment setup complete!"

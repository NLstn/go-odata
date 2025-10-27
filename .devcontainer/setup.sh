#!/bin/bash
set -e

echo "Setting up development environment..."

# Download Go modules
echo "📦 Downloading Go modules..."
go mod download

# Install golangci-lint
echo "🔍 Installing golangci-lint..."
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.5.0

# Install air for hot-reload
echo "🔥 Installing air..."
go install github.com/air-verse/air@latest

# Install wrk for load testing
echo "⚡ Installing wrk..."
sudo apt-get update
sudo apt-get install -y --no-install-recommends wrk

echo "✅ Development environment setup complete!"

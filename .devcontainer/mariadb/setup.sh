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

# Install MariaDB client for debugging
echo "🐬 Installing MariaDB client..."
sudo apt-get update
sudo apt-get install -y --no-install-recommends mariadb-client

echo "✅ Development environment setup complete!"
echo ""
echo "MariaDB connection details:"
echo "  Host: localhost"
echo "  Port: 3306"
echo "  User: odata"
echo "  Password: odata_dev"
echo "  Database: odata_test"
echo ""
echo "Connect with: mariadb -h localhost -u odata -podata_dev odata_test"

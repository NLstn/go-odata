# Dev Container Configuration

This directory contains the development container configuration for GitHub Codespaces and VS Code Dev Containers.

## What's Included

- **Go 1.24**: Latest Go development environment
- **VS Code Extensions**:
  - Go language support with IntelliSense
  - GitHub Copilot for AI-assisted coding
  - GitLens for enhanced Git integration
- **Tools**:
  - golangci-lint for code linting
  - Go language server for enhanced IDE features
  - air for hot-reload development server
- **Port Forwarding**: Port 8080 is automatically forwarded for the development server

## Getting Started

### Using GitHub Codespaces

1. Go to the repository on GitHub
2. Click the "Code" button
3. Select "Codespaces" tab
4. Click "Create codespace on main" (or your branch)
5. Wait for the container to build and start

### Using VS Code Dev Containers

1. Install the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Open the repository in VS Code
3. Press `F1` and select "Dev Containers: Reopen in Container"
4. Wait for the container to build and start

## Running the Development Server

Once the container is running, you can run the development server with hot reload:

```bash
cd cmd/devserver
air
```

Or without hot reload:

```bash
cd cmd/devserver
go run .
```

The server will be available at `http://localhost:8080`.

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...
```

## Code Quality

The container is configured to automatically:
- Format code on save
- Run `go vet` on save
- Organize imports on save
- Use golangci-lint for linting

## Customization

You can customize the dev container by editing `.devcontainer/devcontainer.json`. See the [Dev Containers specification](https://containers.dev/) for more options.

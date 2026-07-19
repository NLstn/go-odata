# MSSQL Dev Container Configuration

This directory contains the MSSQL-based development container configuration for GitHub Codespaces and VS Code Dev Containers.

## What's Included

- **Go 1.25**: Latest Go development environment
- **SQL Server 2022 (Developer edition image)**: SQL Server database for local development and testing
- **SQLite**: Also available for testing (file-based or in-memory)
- **VS Code Extensions**:
  - Go language support with IntelliSense
  - GitHub Copilot for AI-assisted coding
  - GitLens for enhanced Git integration
- **Tools**:
  - golangci-lint for code linting
  - Go language server for enhanced IDE features
  - air for hot-reload development server
  - bombardier for HTTP load testing
- **Port Forwarding**: Port 8080 is automatically forwarded for the development server

## Getting Started

### Using GitHub Codespaces

1. Go to the repository on GitHub
2. Click the "Code" button
3. Select "Codespaces" tab
4. Click "..." (options menu)
5. Select "New with options..."
6. Choose the MSSQL devcontainer configuration
7. Click "Create codespace"
8. Wait for the container to build and start

### Using VS Code Dev Containers

1. Install the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Open the repository in VS Code
3. Press `F1` and select "Dev Containers: Open Folder in Container"
4. Select the `.devcontainer/mssql` folder
5. Wait for the container to build and start

## Database Connection

The SQL Server container is automatically configured with the following credentials:

- **Host**: localhost
- **Port**: 1433
- **User**: sa
- **Password**: StrongPassw0rd!
- **Database**: odata_test

### Connection String

For Go applications using GORM with the SQL Server driver:

```go
dsn := "sqlserver://sa:StrongPassw0rd!@localhost:1433?database=odata_test&encrypt=disable"
```

Or use the environment variable:

```go
dsn := os.Getenv("MSSQL_DSN")
```

### Testing with MSSQL

Run the compliance server with MSSQL:

```bash
cd cmd/complianceserver
go run . -db mssql
```

Run compliance tests against MSSQL:

```bash
cd compliance-suite
go run . -db mssql
```

### Testing with SQLite

SQLite is also available in this container. You can switch between databases:

```bash
# Using SQLite with file-based storage
cd cmd/complianceserver
go run . -db sqlite -dsn /tmp/test.db

# Using SQLite in-memory
cd cmd/complianceserver
go run . -db sqlite -dsn :memory:
```

## Running the Development Server

Once the container is running, you can run the development server with hot reload:

```bash
air
```

Or without hot reload:

```bash
cd cmd/devserver
go run .
```

The `air` command should be run from the repository root. It will automatically watch for changes in the library files and rebuild the development server.

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

## Licensing note

The SQL Server container uses Microsoft's Developer edition image and is intended for local development and CI testing only (non-production usage).

## Customization

You can customize the dev container by editing `.devcontainer/mssql/devcontainer.json`. See the [Dev Containers specification](https://containers.dev/) for more options.

## Alternative Dev Containers

This project also provides devcontainers for other databases:
- PostgreSQL-based devcontainer in the `.devcontainer/postgres` directory
- MySQL-based devcontainer in the `.devcontainer/mysql` directory
- MariaDB-based devcontainer in the `.devcontainer/mariadb` directory

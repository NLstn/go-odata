# MySQL Dev Container Configuration

This directory contains the MySQL-based development container configuration for GitHub Codespaces and VS Code Dev Containers.

## What's Included

- **Go 1.25**: Latest Go development environment
- **MySQL 8**: MySQL database for local development and testing
- **SQLite**: Also available for testing (file-based or in-memory)
- **VS Code Extensions**:
  - Go language support with IntelliSense
  - GitHub Copilot for AI-assisted coding
  - GitLens for enhanced Git integration
- **Tools**:
  - golangci-lint for code linting
  - Go language server for enhanced IDE features
  - air for hot-reload development server
  - wrk for HTTP load testing
  - MySQL client for database debugging
- **Port Forwarding**: Port 8080 is automatically forwarded for the development server

## Getting Started

### Using GitHub Codespaces

1. Go to the repository on GitHub
2. Click the "Code" button
3. Select "Codespaces" tab
4. Click "..." (options menu)
5. Select "New with options..."
6. Choose the MySQL devcontainer configuration
7. Click "Create codespace"
8. Wait for the container to build and start

### Using VS Code Dev Containers

1. Install the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Open the repository in VS Code
3. Press `F1` and select "Dev Containers: Open Folder in Container"
4. Select the `.devcontainer/mysql` folder
5. Wait for the container to build and start

## Database Connection

The MySQL container is automatically configured with the following credentials:

- **Host**: localhost
- **Port**: 3306
- **User**: odata
- **Password**: odata_dev
- **Database**: odata_test

### Connection String

For Go applications using GORM with the MySQL driver:

```go
dsn := "odata:odata_dev@tcp(localhost:3306)/odata_test?parseTime=true"
```

Or use the environment variable:

```go
dsn := os.Getenv("MYSQL_DSN")
```

### Testing with MySQL

Run the compliance server with MySQL:

```bash
cd cmd/complianceserver
go run . -db mysql
```

Run compliance tests against MySQL:

```bash
cd compliance-suite
go run . -db mysql
```

### Using the MySQL Client

Connect to the database directly:

```bash
mysql -h localhost -u odata -podata_dev odata_test
```

Common commands:
```sql
-- Show all tables
SHOW TABLES;

-- Describe a table structure
DESCRIBE products;

-- Query data
SELECT * FROM products LIMIT 10;
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

## Customization

You can customize the dev container by editing `.devcontainer/mysql/devcontainer.json`. See the [Dev Containers specification](https://containers.dev/) for more options.

## Alternative Dev Containers

This project also provides devcontainers for other databases:
- PostgreSQL-based devcontainer in the `.devcontainer/postgres` directory
- MariaDB-based devcontainer in the `.devcontainer/mariadb` directory

## Troubleshooting

### Connection Refused

If you get connection refused errors, the MySQL container may still be starting. Wait a few seconds and try again.

### Permission Denied

If you get permission denied errors when connecting to MySQL, ensure the container has finished starting and the database has been initialized.

### Database Reset

To reset the database to a clean state:

```bash
# Connect to MySQL
mysql -h localhost -u odata -podata_dev odata_test

# Drop and recreate database
DROP DATABASE odata_test;
CREATE DATABASE odata_test;
exit
```

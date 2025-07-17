# GitHub Access Policies Scanner

A gRPC-based backend service that scans GitHub organizations for repository access policies and security violations. The service retrieves detailed information about repositories, collaborators, teams, and branch protection rules, then applies configurable policy rules to identify potential security issues.

## Requirements

- Go 1.23 or later
- GitHub Personal Access Token with appropriate permissions
- Network access to GitHub API

## Setup Instructions

### 1. Clone the Repository

```bash
git clone <repository-url>
cd github-access-policies
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Create Environment Configuration

Copy the example environment file and configure your GitHub token:

```bash
cp .env.example .env
```

Edit `.env` and add your GitHub Personal Access Token:

```bash
GITHUB_TOKEN=your_actual_token_here
```

#### GitHub Token Permissions

Your GitHub Personal Access Token needs the following scopes:

- `repo` - Full control of private repositories
- `read:org` - Read org and team membership, read org projects
  Or run directly with go:

## Development

### Project Structure

```
.
├── api/                     # Protocol buffer definitions
├── gen/                     # Generated gRPC code
├── internal/
│   ├── githubclient/          # GitHub API client
├── .env.example               # Environment variables template
└── README.md                  # This file
```

### Regenerating Protocol Buffers

This project follows semantic versioning for its gRPC API:
When you need to make breaking changes:

1. Create a new version directory:

   ```bash
   mkdir -p api/v2
   mkdir -p gen/v2
   ```

2. Copy the current proto file as a starting point:

   ```bash
   cp api/v1/github_scanner.proto api/v2/github_scanner.proto
   ```

3. Update the package name and go_package option in the new proto file:

   ```proto
   package githubscanner.v2;
   option go_package = "github.com/liadyacobi/github-access-policies/gen/v2";
   ```

4. Make your breaking changes to the v2 proto file

5. Generate the new version using the protoc command:

```bash
protoc --go_out=./gen --go_opt=paths=source_relative \
  --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
  --proto_path=./api ./api/v1/github_scanner.proto
```

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

## Policy Categories

### Policy Input Data Structure

Policies operate on normalized organization data with the following structure:

```json
{
  "name": "organization-name",
  "repositories": {
    "repo-name": {
      "id": 123456,
      "name": "repo-name",
      "full_name": "org/repo-name",
      "private": true,
      "fork": false,
      "archived": false,
      "default_branch": "main"
    }
  },
  "access": {
    "repo-name": {
      "collaborators": {
        "username": {
          "id": 789,
          "login": "username",
          "type": "User",
          "permission": "admin"
        }
      },
      "teams": {
        "team-slug": {
          "id": 111,
          "name": "Team Name",
          "slug": "team-slug",
          "permission": "push"
        }
      }
    }
  },
  "protection": {
    "repo-name": [
      {
        "branch_name": "main",
        "required_approving_review_count": 2,
        "restrict_pushes": true,
        "restricted_push_users": ["admin-user"],
        "restricted_push_teams": ["admin-team"]
      }
    ]
  }
}
```

### Policy Output (Violation) Data Structure

Each violation has the following structure:

```json
{
  "policy_id": string,
  "description": string,
  "severity": "high" | "medium" | "low",
  "repo_name": "myorg/public-repo",
  "details": {
    "key": "value",
    "foo": "bar"
  }
}
```

### Repository Security Policies (`repository.security` package)

These policies ensure repositories follow basic security best practices:

| Policy ID                   | Description                                   | Severity | Details                                           |
| --------------------------- | --------------------------------------------- | -------- | ------------------------------------------------- |
| `REPO_NO_PUBLIC`            | Public repositories are not allowed           | High     | Repository name, current/required visibility      |
| `REPO_NO_MISSING_ADMIN`     | Repositories must have at least one admin     | High     | Repository name, admin count, total collaborators |
| `REPO_NO_RESTRICTED_ADMINS` | Restricted users should not have admin access | High     | Repository name, user, current/max permissions    |

#### REPO_NO_PUBLIC_001

Identifies public repositories

**Example Violation:**

```json
{
  "policy_id": "REPO_NO_PUBLIC_001",
  "description": "Public repositories are not allowed",
  "repo_name": "myorg/public-repo",
  "severity": "high",
  "details": {
    "current_visibility": "public",
    "required_visibility": "private"
  }
}
```

#### REPO_NO_MISSING_ADMIN

The security policies identify several categories of restricted users:

1. **Explicitly Restricted**: `temp-user`, `guest`, `contractor`, `intern`
2. **Temporary Users**: Usernames starting with `temp-` or ending with `-temp`

### Policy Severity Levels

- **High**: Critical security issues that should be addressed immediately

- **Medium**: Important security concerns that should be reviewed

- **Low**: Best practice recommendations and warnings

### Policy Evaluation Flow

1. **Data Collection**: GitHub API client fetches repository, collaborator, team, and branch protection data
2. **Normalization**: Raw GitHub API responses are converted to structured format
3. **Policy Evaluation**: OPA evaluates Rego policies against normalized data
4. **Violation Reporting**: Structured violations are returned via gRPC response

### Custom Policy Development

To add new policies:

1. Create a new `.rego` file in the `policies/` directory
2. Define the package name (e.g., `package custom.policies`)
3. Implement policy rules using the `deny` or `warn` rules
4. Return structured violation objects with unique policy IDs

## Development

### Project Structure

```
.
├── api/                     # Protocol buffer definitions
├── gen/                     # Generated gRPC code
├── internal/
│   ├── githubclient/          # GitHub API client
│   ├── normalizer/            # Data normalization layer
├── .env.example               # Environment variables template
└── README.md                  # This file
```

### Key Components

1. **GitHub Client** (`internal/githubclient/`):

   - Pure API client with pagination support
   - Returns raw GitHub API responses
   - Handles authentication and rate limiting

2. **Normalizer** (`internal/normalizer/`):
   - Converts GitHub API responses to structured data
   - No external dependencies or API calls
   - Provides consistent data format for further processing

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

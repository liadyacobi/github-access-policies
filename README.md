# GitHub Access Policies Scanner

A gRPC-based backend service that scans GitHub organizations for repository access policies and security violations. The service retrieves detailed information about repositories, collaborators, teams, and branch protection rules, then applies configurable policy rules using Open Policy Agent (OPA) to identify potential security issues.

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
PORT=50001
POLICY_DIR=./policies
```

#### GitHub Token Permissions

Your GitHub Personal Access Token needs the following scopes:

- `repo` - Full control of private repositories
- `read:org` - Read org and team membership, read org projects

### 4. Run the Service

You can run the service directly with go:

```bash
go run cmd/server/main.go
```

Or build and run the binary:

```bash
go build -o github-scanner cmd/server/main.go
GITHUB_TOKEN=your_actual_token_here PORT=50001 POLICY_DIR=./policies ./github-scanner
```

The service will start on the specified port (default it 50001 if not specified).

## Usage

### gRPC Service Endpoints

The service exposes the following gRPC endpoints:

#### GetRepositories

Scans all repositories in a GitHub organization and returns access information with policy violations.

**Request:**

```proto
message GetRepositoriesRequest {
  string organization_name = 1;
}
```

**Response:**

```proto
message GetRepositoriesResponse {
  repeated RepositoryScanResult repositories = 1;
}
```

### Sample gRPC Request

Using grpcurl (install with `go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest`):

```bash
grpcurl -plaintext -d '{"organization_name": "your-org"}' \
  localhost:50001 githubscanner.v1.GithubScanner/GetRepositories
```

### Sample Response

```json
{
  "repositories": [
    {
      "id": "123456",
      "fullName": "myorg/my-repo",
      "htmlUrl": "https://github.com/myorg/my-repo",
      "isPublic": true,
      "isFork": false,
      "collaborators": [
        {
          "githubId": "789",
          "login": "john.doe",
          "type": "User",
          "permission": "admin"
        },
        {
          "githubId": "790",
          "login": "temp-johntest",
          "type": "User",
          "permission": "admin"
        }
      ],
      "teamAccesses": [
        {
          "teamId": "111",
          "teamName": "Developers",
          "slug": "developers",
          "permission": "push"
        }
      ],
      "branchProtectionRules": [
        {
          "branchName": "main",
          "requiredApprovingReviewCount": 2,
          "restrictPushes": true,
          "restrictedPushUsers": ["admin-user"],
          "restrictedPushTeams": ["admin-team"]
        }
      ],
      "violations": [
        {
          "policyId": "REPO_NO_PUBLIC",
          "description": "Public repositories are not allowed",
          "severity": "high",
          "details": {
            "current_visibility": "public",
            "required_visibility": "private"
          }
        },
        {
          "policyId": "REPO_NO_RESTRICTED_ADMINS",
          "description": "Restricted users should not have admin access",
          "severity": "high",
          "details": {
            "user": "temp-johntest",
            "current_permission": "admin",
            "max_allowed_permission": "pull"
          }
        }
      ]
    }
  ]
}
```

## Policy Engine

The service uses **Open Policy Agent (OPA)** with Rego policies to evaluate repository access and security configurations. Policies are loaded from the `policies/` directory at startup.

### Policy Configuration

Policies are defined in Rego files in the `policies/` directory. The service automatically loads all `.rego` files from this directory.

### Current Policy Categories

#### Repository Security Policies (`repository.security` package)

These policies ensure repositories follow basic security best practices:

| Policy ID                   | Description                                   | Severity | Details                                           |
| --------------------------- | --------------------------------------------- | -------- | ------------------------------------------------- |
| `REPO_NO_PUBLIC`            | Public repositories are not allowed           | High     | Repository name, current/required visibility      |
| `REPO_NO_MISSING_ADMIN`     | Repositories must have at least one admin     | High     | Repository name, admin count, total collaborators |
| `REPO_NO_RESTRICTED_ADMINS` | Restricted users should not have admin access | High     | Repository name, user, current/max permissions    |

#### REPO_NO_PUBLIC

Identifies public repositories that should be private.

**Example Violation:**

```json
{
  "policy_id": "REPO_NO_PUBLIC",
  "description": "Public repositories are not allowed",
  "severity": "high",
  "details": {
    "current_visibility": "public",
    "required_visibility": "private"
  }
}
```

#### REPO_NO_MISSING_ADMIN

Ensures repositories have at least one admin user.

#### REPO_NO_RESTRICTED_ADMINS

The security policies identify several categories of restricted users:

1. **Explicitly Restricted**: `temp-user`, `guest`, `contractor`, `intern`
2. **Temporary Users**: Usernames starting with `temp-` or ending with `-temp`

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

### Adding Custom Policies

To add new policies:

1. Create a new `.rego` file in the `policies/` directory
2. Define the package name (e.g., `package custom.policies`)
3. Implement policy rules using the `deny` rule pattern
4. Return structured violation objects with unique policy IDs

Example custom policy:

```rego
package custom.policies

import rego.v1

# Policy: Repositories must have branch protection on main branch
deny contains violation if {
    repo_name := input.repositories[_].name
    repo := input.repositories[repo_name]
    protection := input.protection[repo_name]

    # Check if main branch exists but has no protection
    not has_main_protection(protection)

    violation := {
        "policy_id": "CUSTOM_BRANCH_PROTECTION",
        "repo_name": repo_name,
        "description": "Main branch must have protection enabled",
        "severity": "medium",
        "details": {
            "branch": "main",
            "protection_status": "disabled"
        }
    }
}

has_main_protection(protection) if {
    protection[_].branch_name == "main"
}
```

## Environment Variables

The service supports the following environment variables:

| Variable       | Description                       | Default    | Required |
| -------------- | --------------------------------- | ---------- | -------- |
| `GITHUB_TOKEN` | GitHub Personal Access Token      | -          | Yes      |
| `PORT`         | gRPC server port                  | 50001      | No       |
| `POLICY_DIR`   | Directory containing policy files | ./policies | No       |

### Example .env File

```bash
# GitHub Personal Access Token (required)
GITHUB_TOKEN=ghp_your_token_here

# gRPC server port (optional, defaults to 50001)
PORT=50001

# Policy directory (optional, defaults to ./policies)
POLICY_DIR=./policies
```

## Development

### Project Structure

```
.
├── api/                     # Protocol buffer definitions
│   └── v1/                  # API version 1
├── gen/                     # Generated gRPC code
│   └── v1/                  # Generated Go code for API v1
├── cmd/
│   └── server/              # Server main entry point
├── internal/
│   ├── githubclient/        # GitHub API client
│   ├── normalizer/          # Data normalization layer
│   ├── policy/              # Policy engine (OPA integration)
│   └── server/              # gRPC server implementation
├── policies/                # Rego policy files
│   └── repository_security.rego
├── .env.example             # Environment variables template
└── README.md                # This file
```

### Key Components

1. **GitHub Client** (`internal/githubclient/`):

   - Pure API client with pagination support
   - Returns raw GitHub API responses
   - Handles authentication and rate limiting

2. **Normalizer** (`internal/normalizer/`):

   - Converts GitHub API responses to structured data
   - No external dependencies or API calls
   - Provides consistent data format for policy evaluation

3. **Policy Engine** (`internal/policy/`):

   - OPA integration for policy evaluation
   - Loads Rego policies from filesystem
   - Returns structured violation

4. **gRPC Server** (`internal/server/`):
   - Orchestrates data collection and policy evaluation
   - Implements the gRPC service interface
   - Handles request/response conversion

### Building and Testing

Build the service:

```bash
go build -o github-scanner cmd/server/main.go
```

### Regenerating Protocol Buffers

If you need to modify the gRPC API:

```bash
protoc --go_out=./gen --go_opt=paths=source_relative \
  --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
  --proto_path=./api ./api/v1/github_scanner.proto
```

For breaking changes, create a new API version:

1. Create new directories: `api/v2/` and `gen/v2/`
2. Copy and modify the proto file
3. Update package names and imports
4. Generate new code

## Troubleshooting

### Common Issues

1. **"GITHUB_TOKEN environment variable is required"**

   - Ensure your `.env` file contains a valid GitHub token
   - Check that the token has the required scopes (classic token)

2. **"failed to list repositories"**

   - Verify the organization name is correct
   - Check that your token has access to the organization

3. **"policy directory does not exist"**

   - Ensure the `policies/` directory exists
   - Check the `POLICY_DIR` environment variable

4. **gRPC connection issues**
   - Verify the service is running on the expected port
   - Check firewall settings if accessing remotely

### Performance Considerations

- The service fetches detailed information for each repository sequentially
- Large organizations may experience longer response times
- Consider implementing concurrent fetching for better performance

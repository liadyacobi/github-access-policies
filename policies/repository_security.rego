package repository.security

import rego.v1

# Policy: Public repositories are not allowed
deny contains violation if {
    repo_name := input.repositories[_].name
    repo := input.repositories[repo_name]
    repo.private == false
    violation := {
        "policy_id": "REPO_NO_PUBLIC_001",
        "repo_name": repo_name,
        "description": "Public repositories are not allowed",
        "severity": "high",
        "details": {
            "current_visibility": "public",
            "required_visibility": "private"
        }
    }
}

# Policy: Repositories must have at least one admin
deny contains violation if {
    min_admins := 1
    repo_name := input.repositories[_].name
    repo := input.repositories[repo_name]
    access := input.access[repo_name]
    admins := [collab | collab := access.collaborators[_]; collab.permission == "admin"]
    count(admins) < min_admins
    violation := {
        "policy_id": "REPO_NO_MISSING_ADMIN",
        "repo_name": repo_name,
        "description": "Repository does not have enough admins",
        "severity": "high",
        "details": {
            "admin_count": sprintf("%d", [count(admins)]),
            "required_min_admins": sprintf("%d", [min_admins]),
            "total_collaborators": sprintf("%d", [count(access.collaborators)])
        }
    }
}

# Policy: Certain users should not have admin access
deny contains violation if {
    repo_name := input.repositories[_].name
    repo := input.repositories[repo_name]
    access := input.access[repo_name]
    collaborator := access.collaborators[_]
    collaborator.permission == "admin"
    is_restricted_user(collaborator.login)
    violation := {
        "policy_id": "REPO_NO_RESTRICTED_ADMINS",
        "repo_name": repo_name,
        "description": "Restricted users should not have admin access",
        "severity": "high",
        "details": {
            "user": collaborator.login,
            "current_permission": collaborator.permission,
            "max_allowed_permission": "pull"
        }
    }
}

# Helper functions
is_restricted_user(username) if {
    username in ["temp-user", "guest", "contractor", "intern", "liadyacobi"]
}

is_restricted_user(username) if {
    startswith(username, "temp-")
}

is_restricted_user(username) if {
    endswith(username, "-temp")
}
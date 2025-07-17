package normalizer

import (
	"github.com/google/go-github/v57/github"
)

type RepositoryData struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	Fork          bool   `json:"fork"`
	Archived      bool   `json:"archived"`
	DefaultBranch string `json:"default_branch"`
}

type CollaboratorData struct {
	ID         int64  `json:"id"`
	Login      string `json:"login"`
	Type       string `json:"type"`
	Permission string `json:"permission"`
}

type TeamData struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Permission string `json:"permission"`
}

type BranchProtectionData struct {
	BranchName                   string   `json:"branch_name"`
	RequiredApprovingReviewCount int      `json:"required_approving_review_count"`
	RestrictPushes               bool     `json:"restrict_pushes"`
	RestrictedPushUsers          []string `json:"restricted_push_users"`
	RestrictedPushTeams          []string `json:"restricted_push_teams"`
}

type RepositoryCompleteData struct {
	Repository RepositoryData         `json:"repository"`
	Access     AccessData             `json:"access"`
	Protection []BranchProtectionData `json:"protection"`
}

type OrganizationData struct {
	Name         string                            `json:"name"`
	Repositories map[string]RepositoryData         `json:"repositories"`
	Access       map[string]AccessData             `json:"access"`     // repo_name -> access data
	Protection   map[string][]BranchProtectionData `json:"protection"` // repo_name -> branch protections
}

type AccessData struct {
	Collaborators map[string]CollaboratorData `json:"collaborators"` // login -> collaborator data
	Teams         map[string]TeamData         `json:"teams"`         // slug -> team data
}

func normalizeRepository(repo *github.Repository) RepositoryData {
	return RepositoryData{
		ID:            repo.GetID(),
		Name:          repo.GetName(),
		FullName:      repo.GetFullName(),
		Private:       repo.GetPrivate(),
		Fork:          repo.GetFork(),
		Archived:      repo.GetArchived(),
		DefaultBranch: repo.GetDefaultBranch(),
	}
}

func normalizeCollaborator(user *github.User, permission *github.RepositoryPermissionLevel) CollaboratorData {
	perm := "none"
	if permission != nil {
		perm = permission.GetPermission()
	}

	return CollaboratorData{
		ID:         user.GetID(),
		Login:      user.GetLogin(),
		Type:       user.GetType(),
		Permission: perm,
	}
}

func normalizeTeam(team *github.Team) TeamData {
	return TeamData{
		ID:         team.GetID(),
		Name:       team.GetName(),
		Slug:       team.GetSlug(),
		Permission: team.GetPermission(),
	}
}

func normalizeBranchProtection(branchName string, protection *github.Protection) BranchProtectionData {
	data := BranchProtectionData{
		BranchName:                   branchName,
		RequiredApprovingReviewCount: 0,
		RestrictPushes:               false,
		RestrictedPushUsers:          []string{},
		RestrictedPushTeams:          []string{},
	}

	if protection == nil {
		return data
	}

	if reviews := protection.GetRequiredPullRequestReviews(); reviews != nil {
		data.RequiredApprovingReviewCount = reviews.RequiredApprovingReviewCount
	}

	if restrictions := protection.GetRestrictions(); restrictions != nil {
		data.RestrictPushes = true

		for _, user := range restrictions.Users {
			data.RestrictedPushUsers = append(data.RestrictedPushUsers, user.GetLogin())
		}

		for _, team := range restrictions.Teams {
			data.RestrictedPushTeams = append(data.RestrictedPushTeams, team.GetSlug())
		}
	}

	return data
}

func normalizeRepositoryData(
	repo *github.Repository,
	collaborators []*github.User,
	permissions map[string]*github.RepositoryPermissionLevel,
	teams []*github.Team,
	branchProtections map[string]*github.Protection,
) RepositoryCompleteData {

	repoData := normalizeRepository(repo)

	// Access data
	accessData := AccessData{
		Collaborators: make(map[string]CollaboratorData),
		Teams:         make(map[string]TeamData),
	}

	// Normalize collaborators
	for _, collab := range collaborators {
		login := collab.GetLogin()
		permission := permissions[login]
		accessData.Collaborators[login] = normalizeCollaborator(collab, permission)
	}

	// Normalize teams
	for _, team := range teams {
		accessData.Teams[team.GetSlug()] = normalizeTeam(team)
	}

	// Normalize branch protections
	var protections []BranchProtectionData
	for branchName, protection := range branchProtections {
		protections = append(protections, normalizeBranchProtection(branchName, protection))
	}

	return RepositoryCompleteData{
		Repository: repoData,
		Access:     accessData,
		Protection: protections,
	}
}

// NormalizeOrganizationData creates a comprehensive normalized view of an organization
func NormalizeOrganizationData(
	orgName string,
	repos []*github.Repository,
	collaborators map[string][]*github.User,
	permissions map[string]map[string]*github.RepositoryPermissionLevel,
	teams map[string][]*github.Team,
	branchProtections map[string]map[string]*github.Protection,
) OrganizationData {

	orgData := OrganizationData{
		Name:         orgName,
		Repositories: make(map[string]RepositoryData),
		Access:       make(map[string]AccessData),
		Protection:   make(map[string][]BranchProtectionData),
	}

	// Process each repository using NormalizeRepositoryData
	for _, repo := range repos {
		repoName := repo.GetName()

		// Get data for this repository
		repoCollaborators := collaborators[repoName]
		repoPermissions := permissions[repoName]
		repoTeams := teams[repoName]
		repoBranchProtections := branchProtections[repoName]

		// Normalize this repository's data
		normalizedRepo := normalizeRepositoryData(
			repo,
			repoCollaborators,
			repoPermissions,
			repoTeams,
			repoBranchProtections,
		)

		orgData.Repositories[repoName] = normalizedRepo.Repository
		orgData.Access[repoName] = normalizedRepo.Access
		orgData.Protection[repoName] = normalizedRepo.Protection
	}

	return orgData
}

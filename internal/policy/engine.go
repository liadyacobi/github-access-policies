package policy

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/liadyacobi/github-access-policies/internal/normalizer"
	"github.com/open-policy-agent/opa/v1/rego"
)

// PolicyViolation represents a policy violation found during evaluation
type PolicyViolation struct {
	PolicyID    string            `json:"policy_id"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Details     map[string]string `json:"details"`
}

// PolicyEngine handles policy evaluation using OPA
type PolicyEngine struct {
	preparedQuery *rego.PreparedEvalQuery
}

// NewPolicyEngine creates a new policy engine by loading policies from a directory
// This function can be extendible to load policies from different sources in the future
func NewPolicyEngine(policyDir string) (*PolicyEngine, error) {
	// Check if directory exists
	if _, err := os.Stat(policyDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("policy directory does not exist: %s", policyDir)
	}

	// Load all .rego files from the directory
	r := rego.New(
		rego.Load([]string{policyDir}, nil),
		rego.Query("data"), // Query everything
	)

	// Prepare the query for evaluation
	preparedQuery, err := r.PrepareForEval(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to prepare policies: %w", err)
	}

	log.Printf("Successfully loaded policies from directory: %s", policyDir)

	return &PolicyEngine{
		preparedQuery: &preparedQuery,
	}, nil
}

// EvaluateOrganization evaluates the organization data against all loaded policies
func (pe *PolicyEngine) EvaluateOrganization(ctx context.Context, orgData normalizer.OrganizationData) ([]PolicyViolation, error) {
	log.Printf("Evaluating organization: %s", orgData.Name)

	var allViolations []PolicyViolation

	// Evaluate repository security policies
	repoViolations, err := pe.preparedQuery.Eval(ctx, rego.EvalInput(orgData))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate repository security policies: %w", err)
	}

	for _, v := range repoViolations {
		violation, err := pe.parseViolations(v)
		if err != nil {
			log.Printf("Warning: failed to parse violation: %v", err)
			continue
		}
		allViolations = append(allViolations, violation...)
	}
	log.Printf("Repository security policy evaluation returned %d violations", len(repoViolations))

	return allViolations, nil
}

// parseViolations converts OPA result to PolicyViolation structs
func (pe *PolicyEngine) parseViolations(v rego.Result) ([]PolicyViolation, error) {
	if len(v.Expressions) == 0 {
		return nil, fmt.Errorf("no expressions found in OPA result")
	}

	// Currently evaluating all as a single expression
	expr := v.Expressions[0]
	if expr.Value == nil {
		return nil, fmt.Errorf("expression value is nil")
	}

	// Handle different possible structures from OPA
	data, ok := expr.Value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expression value is not of type map[string]interface{}")
	}
	violations := pe.extractViolationsFromMap(data)

	return violations, nil
}

// extractViolationsFromMap recursively extracts violations from nested map structure
func (pe *PolicyEngine) extractViolationsFromMap(data map[string]interface{}) []PolicyViolation {
	var violations []PolicyViolation

	for _, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively search nested maps
			violations = append(violations, pe.extractViolationsFromMap(v)...)
		case []interface{}:
			// Found an array - check if it contains violations
			for _, item := range v {
				if violationMap, ok := item.(map[string]interface{}); ok {
					// Check if this looks like a violation (has required fields)
					if pe.isViolationMap(violationMap) {
						violation, err := pe.parseViolation(violationMap)
						if err != nil {
							log.Printf("Warning: failed to parse violation: %v", err)
							continue // Skip invalid violations
						}
						violations = append(violations, violation)
					}
				}
			}
		}
	}

	return violations
}

// isViolationMap checks if a map contains the required fields for a violation
func (pe *PolicyEngine) isViolationMap(m map[string]interface{}) bool {
	// Check for required violation fields
	_, hasPolicyID := m["policy_id"]
	_, hasDescription := m["description"]
	_, hasSeverity := m["severity"]

	return hasPolicyID && hasDescription && hasSeverity
}

// parseViolation converts a single violation map to PolicyViolation struct
func (pe *PolicyEngine) parseViolation(violationMap map[string]interface{}) (PolicyViolation, error) {
	// Safely extract required fields
	policyID, ok := violationMap["policy_id"].(string)
	if !ok {
		return PolicyViolation{}, fmt.Errorf("policy_id is missing or not a string")
	}

	description, ok := violationMap["description"].(string)
	if !ok {
		return PolicyViolation{}, fmt.Errorf("description is missing or not a string")
	}

	severity, ok := violationMap["severity"].(string)
	if !ok {
		return PolicyViolation{}, fmt.Errorf("severity is missing or not a string")
	}

	violation := PolicyViolation{
		PolicyID:    policyID,
		Description: description,
		Severity:    severity,
		Details:     make(map[string]string),
	}

	// Safely extract details if they exist
	if detailsData, exists := violationMap["details"]; exists {
		if detailsMap, ok := detailsData.(map[string]interface{}); ok {
			for k, v := range detailsMap {
				if strValue, ok := v.(string); ok {
					violation.Details[k] = strValue
				}
			}
		}
	}

	return violation, nil
}

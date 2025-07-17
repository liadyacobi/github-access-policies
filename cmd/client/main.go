package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pb "github.com/liadyacobi/github-access-policies/gen/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var (
		serverAddr = flag.String("server", "localhost:50001", "The server address in the format host:port")
		orgName    = flag.String("org", "", "GitHub organization name (required)")
		timeout    = flag.Duration("timeout", 30*time.Second, "Request timeout")
	)
	flag.Parse()

	if *orgName == "" {
		log.Fatal("Organization name is required. Use -org flag")
	}

	// Set up a connection to the server
	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Create a client
	client := pb.NewGithubScannerClient(conn)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Make the request
	log.Printf("Scanning organization: %s", *orgName)
	log.Printf("Connecting to server: %s", *serverAddr)

	request := &pb.GetRepositoriesRequest{
		OrganizationName: *orgName,
	}

	response, err := client.GetRepositories(ctx, request)
	if err != nil {
		log.Fatalf("Failed to get repositories: %v", err)
	}

	// Print the results
	printResults(response)
}

func printResults(response *pb.GetRepositoriesResponse) {
	fmt.Printf("\n=== SCAN RESULTS ===\n")
	fmt.Printf("Found %d repositories\n\n", len(response.Repositories))

	for i, repo := range response.Repositories {
		fmt.Printf("--- Repository %d ---\n", i+1)
		fmt.Printf("Name: %s\n", repo.FullName)
		fmt.Printf("URL: %s\n", repo.HtmlUrl)
		fmt.Printf("Public: %v\n", repo.IsPublic)
		fmt.Printf("Fork: %v\n", repo.IsFork)

		// Print collaborators
		fmt.Printf("Collaborators (%d):\n", len(repo.Collaborators))
		for _, collab := range repo.Collaborators {
			fmt.Printf("  - %s (%s) - %s\n", collab.Login, collab.Type, collab.Permission)
		}

		// Print team access
		fmt.Printf("Team Access (%d):\n", len(repo.TeamAccesses))
		for _, team := range repo.TeamAccesses {
			fmt.Printf("  - %s (%s) - %s\n", team.TeamName, team.Slug, team.Permission)
		}

		// Print branch protection rules
		fmt.Printf("Branch Protection Rules (%d):\n", len(repo.BranchProtectionRules))
		for _, rule := range repo.BranchProtectionRules {
			fmt.Printf("  - %s: %d required reviews, push restrictions: %v\n",
				rule.BranchName, rule.RequiredApprovingReviewCount, rule.RestrictPushes)
		}

		// Print violations
		fmt.Printf("Policy Violations (%d):\n", len(repo.Violations))
		for _, violation := range repo.Violations {
			fmt.Printf("  - [%s] %s: %s\n", violation.Severity, violation.PolicyId, violation.Description)
			if violation.Details != nil {
				fmt.Printf("    Details: %v\n", violation.Details)
			}
		}

		fmt.Println()
	}

	// Print summary
	totalViolations := 0
	for _, repo := range response.Repositories {
		totalViolations += len(repo.Violations)
	}
	fmt.Printf("=== SUMMARY ===\n")
	fmt.Printf("Total repositories scanned: %d\n", len(response.Repositories))
	fmt.Printf("Total policy violations: %d\n", totalViolations)
}

#!/bin/bash

# Set your environment variables
export GITHUB_TOKEN="your_github_token_here" # Replace with your GitHub token
export POLICY_DIR="./policies"
export PORT="50001"
export ORG_NAME="your_org_name_here" # Replace with your organization name

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting end-to-end test...${NC}"

# Check if GitHub token is set
if [ -z "$GITHUB_TOKEN" ] || [ "$GITHUB_TOKEN" = "your_github_token_here" ]; then
    echo -e "${RED}Error: Please set your GITHUB_TOKEN in the script${NC}"
    exit 1
fi

# Check if OrgName is set
if [ -z "$ORG_NAME" ] || [ "$ORG_NAME" = "your_org_name_here" ]; then
    echo -e "${RED}Error: Please set your ORG_NAME in the script${NC}"
    exit 1
fi

# Build the server and client
echo -e "${YELLOW}Building server...${NC}"
go build -o bin/server ./cmd/server

echo -e "${YELLOW}Building client...${NC}"
go build -o bin/client ./cmd/client

# Start the server in background
echo -e "${YELLOW}Starting server...${NC}"
./bin/server &
SERVER_PID=$!

# Wait for server to start
echo -e "${YELLOW}Waiting for server to start...${NC}"
sleep 3

# Test with a GitHub organization (replace with your test org)
echo -e "${YELLOW}Testing with organization: $ORG_NAME${NC}"

# Run the client
echo -e "${GREEN}Running client...${NC}"
./bin/client -org "$ORG_NAME" -timeout 60s

# Cleanup
echo -e "${YELLOW}Cleaning up...${NC}"
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo -e "${GREEN}Test completed!${NC}"
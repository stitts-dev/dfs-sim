#!/bin/bash

echo "Testing backend API endpoints..."

# Test health endpoint
echo -e "\n1. Testing health endpoint:"
curl -s http://localhost:8080/health | jq .

# Test preferences endpoint (will fail without auth, but we can see if it's reachable)
echo -e "\n2. Testing preferences endpoint (should return 401 Unauthorized):"
curl -s -w "\nHTTP Status: %{http_code}\n" http://localhost:8080/api/v1/user/preferences

# Test glossary endpoint (public)
echo -e "\n3. Testing glossary endpoint:"
curl -s http://localhost:8080/api/v1/glossary | jq . | head -20

# Test contests endpoint (public)
echo -e "\n4. Testing contests endpoint:"
curl -s http://localhost:8080/api/v1/contests | jq .
#!/bin/bash

# Test runner script for v3 lineup generation
# This script runs all tests and provides detailed output

echo "Running tests for v3 lineup generation..."
echo "========================================"

# Change to the v3 directory
cd "$(dirname "$0")"

# Run tests with verbose output
echo "Running algorithm tests..."
go test -v ./tests/

echo ""
echo "Running all tests..."
go test -v ./tests/

echo ""
echo "Running tests with coverage..."
go test -v -cover ./tests/

echo ""
echo "Running benchmark tests..."
go test -v -bench=. ./tests/

echo ""
echo "Test run complete!"

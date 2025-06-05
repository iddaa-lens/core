#!/bin/bash

# Script to migrate git configuration to new iddaa-lens/core repository

# Change to the project directory
cd "$(dirname "$0")"

echo "Setting up git configuration for iddaa-lens/core..."

# Remove existing git remote
git remote remove origin 2>/dev/null || true

# Add new remote
git remote add origin https://github.com/iddaa-lens/core.git

# Verify remote
echo "Current git remote:"
git remote -v

# Update go.mod to reflect new module path
echo "Updating go.mod module path..."
sed -i '' 's|github.com/betslib/iddaa-core|github.com/iddaa-lens/core|g' go.mod

# Update all import statements in Go files
echo "Updating import statements..."
find . -name "*.go" -type f -exec sed -i '' 's|github.com/betslib/iddaa-core|github.com/iddaa-lens/core|g' {} \;

echo "Migration script completed!"
echo "Next steps:"
echo "1. cd /Users/yetkin/dev/iddaa-lens/core"
echo "2. git add ."
echo "3. git commit -m 'Migrate to iddaa-lens organization'"
echo "4. git push -u origin main"
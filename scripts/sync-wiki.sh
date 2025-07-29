#!/bin/bash

# Sync documentation from docs/ directory to GitHub Wiki
# This script uses git subtree to push changes to the wiki repository

set -e

echo "🔄 Syncing documentation to GitHub Wiki..."

# Ensure we have the wiki remote
if ! git remote get-url wiki >/dev/null 2>&1; then
    echo "Adding wiki remote..."
    git remote add wiki https://github.com/nhalm/skimatik.wiki.git
fi

# Create the first wiki page if the wiki doesn't exist
echo "📝 Creating initial wiki page..."
if ! git ls-remote --exit-code wiki master >/dev/null 2>&1; then
    echo "Wiki repository doesn't exist yet - it will be created on first push"
fi

# Push docs directory to wiki using subtree
echo "🚀 Pushing docs/ to wiki..."
git subtree push --prefix=docs wiki master

echo "✅ Documentation successfully synced to GitHub Wiki!"
echo "📖 View at: https://github.com/nhalm/skimatik/wiki" 
#!/usr/bin/env bash
set -euo pipefail

# Release script for Obeya CLI
# Creates a git tag and pushes it, triggering the GitHub Actions release pipeline.
# The pipeline runs GoReleaser which builds binaries, creates a GitHub Release,
# and pushes the Homebrew formula to schoolofai/homebrew-tap.
#
# Usage:
#   ./scripts/release.sh 0.2.0
#   ./scripts/release.sh 0.2.0 "feat: shared board support"

VERSION="${1:-}"
MESSAGE="${2:-}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version> [message]"
    echo "  version: semver without 'v' prefix (e.g., 0.1.0, 1.0.0)"
    echo "  message: optional tag message (default: 'Release v<version>')"
    echo ""
    echo "Examples:"
    echo "  $0 0.1.0"
    echo "  $0 0.2.0 \"feat: shared board support\""
    exit 1
fi

# Strip 'v' prefix if accidentally included
VERSION="${VERSION#v}"
TAG="v${VERSION}"

if [ -z "$MESSAGE" ]; then
    MESSAGE="Release ${TAG}"
fi

# Preflight checks
echo "=== Preflight checks ==="

# 1. Clean working tree
if [ -n "$(git status --porcelain)" ]; then
    echo "ERROR: Working tree is not clean. Commit or stash changes first."
    git status --short
    exit 1
fi
echo "[ok] Working tree clean"

# 2. On main branch
BRANCH=$(git branch --show-current)
if [ "$BRANCH" != "main" ]; then
    echo "ERROR: Not on main branch (currently on '${BRANCH}'). Switch to main first."
    exit 1
fi
echo "[ok] On main branch"

# 3. Up to date with remote
git fetch origin main --quiet
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main)
if [ "$LOCAL" != "$REMOTE" ]; then
    AHEAD=$(git rev-list origin/main..HEAD --count)
    BEHIND=$(git rev-list HEAD..origin/main --count)
    if [ "$BEHIND" -gt 0 ]; then
        echo "ERROR: Local is ${BEHIND} commit(s) behind origin/main. Pull first."
        exit 1
    fi
    if [ "$AHEAD" -gt 0 ]; then
        echo "WARNING: Local is ${AHEAD} commit(s) ahead of origin/main."
        read -rp "Push to main before tagging? [Y/n] " PUSH_ANSWER
        if [ "${PUSH_ANSWER:-Y}" != "n" ] && [ "${PUSH_ANSWER:-Y}" != "N" ]; then
            git push origin main
            echo "[ok] Pushed to main"
        else
            echo "ERROR: Cannot tag without pushing. Remote must have these commits."
            exit 1
        fi
    fi
fi
echo "[ok] In sync with origin/main"

# 4. Tag doesn't already exist
if git tag -l "$TAG" | grep -q "$TAG"; then
    echo "ERROR: Tag '${TAG}' already exists."
    exit 1
fi
echo "[ok] Tag ${TAG} is available"

# 5. Tests pass
echo ""
echo "=== Running tests ==="
go test ./...
echo "[ok] Tests pass"

# Create and push tag
echo ""
echo "=== Creating release ==="
echo "Tag:     ${TAG}"
echo "Message: ${MESSAGE}"
echo ""
read -rp "Create and push tag? [Y/n] " CONFIRM
if [ "${CONFIRM:-Y}" = "n" ] || [ "${CONFIRM:-Y}" = "N" ]; then
    echo "Aborted."
    exit 0
fi

git tag -a "$TAG" -m "$MESSAGE"
git push origin "$TAG"

echo ""
echo "=== Release triggered ==="
echo "Tag ${TAG} pushed. GitHub Actions will now:"
echo "  1. Build binaries for darwin/linux/windows (amd64 + arm64)"
echo "  2. Create GitHub Release with artifacts"
echo "  3. Push Homebrew formula to schoolofai/homebrew-tap"
echo ""
echo "Watch the pipeline: https://github.com/schoolofai/obeya/actions"
echo ""
echo "Once complete, users can install with:"
echo "  brew tap schoolofai/tap"
echo "  brew install obeya"

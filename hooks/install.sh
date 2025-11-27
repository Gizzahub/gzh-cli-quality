#!/bin/bash
# Install gzh-cli-quality Git hooks
#
# Usage:
#   bash hooks/install.sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}📦 Installing gzh-cli-quality Git hooks...${NC}"

# Check if we're in a git repository
if [ ! -d ".git" ]; then
    echo -e "${RED}❌ Error: Not a git repository${NC}"
    echo -e "${YELLOW}Run this script from the root of your git repository${NC}"
    exit 1
fi

# Check if gz-quality is installed
if ! command -v gz-quality &> /dev/null; then
    echo -e "${YELLOW}⚠️  gz-quality not found in PATH${NC}"
    echo -e "${YELLOW}Install it with:${NC}"
    echo -e "  go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest"
    read -p "Continue installation anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Create hooks directory if it doesn't exist
mkdir -p .git/hooks

# Install pre-commit hook
HOOK_SOURCE="hooks/pre-commit"
HOOK_TARGET=".git/hooks/pre-commit"

if [ ! -f "$HOOK_SOURCE" ]; then
    echo -e "${RED}❌ Error: $HOOK_SOURCE not found${NC}"
    exit 1
fi

# Backup existing hook if present
if [ -f "$HOOK_TARGET" ]; then
    BACKUP="$HOOK_TARGET.backup.$(date +%Y%m%d_%H%M%S)"
    echo -e "${YELLOW}⚠️  Existing pre-commit hook found${NC}"
    echo -e "${YELLOW}Backing up to: $BACKUP${NC}"
    mv "$HOOK_TARGET" "$BACKUP"
fi

# Copy and make executable
cp "$HOOK_SOURCE" "$HOOK_TARGET"
chmod +x "$HOOK_TARGET"

echo -e "${GREEN}✅ Pre-commit hook installed successfully!${NC}"
echo
echo -e "${BLUE}Hook location:${NC} $HOOK_TARGET"
echo
echo -e "${BLUE}Configuration:${NC}"
echo -e "  Set GZ_QUALITY_MODE environment variable to customize behavior:"
echo -e "    export GZ_QUALITY_MODE=check    ${YELLOW}# Lint only (default)${NC}"
echo -e "    export GZ_QUALITY_MODE=format   ${YELLOW}# Format only${NC}"
echo -e "    export GZ_QUALITY_MODE=run      ${YELLOW}# Format + lint${NC}"
echo
echo -e "${BLUE}Usage:${NC}"
echo -e "  git commit               ${YELLOW}# Hook runs automatically${NC}"
echo -e "  git commit --no-verify   ${YELLOW}# Skip hook${NC}"
echo
echo -e "${BLUE}Uninstall:${NC}"
echo -e "  rm $HOOK_TARGET"
echo
echo -e "${GREEN}Happy coding! 🚀${NC}"

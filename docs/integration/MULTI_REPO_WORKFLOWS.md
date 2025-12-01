# Multi-Repository Workflows

Patterns and best practices for managing code quality across multiple repositories with gz-quality.

## Table of Contents

- [Overview](#overview)
- [Repository Types](#repository-types)
- [Workflow Patterns](#workflow-patterns)
- [Configuration Management](#configuration-management)
- [CI/CD Strategies](#cicd-strategies)
- [Tool Installation](#tool-installation)
- [Reporting and Aggregation](#reporting-and-aggregation)
- [Real-World Examples](#real-world-examples)
- [Best Practices](#best-practices)

---

## Overview

### Challenges

Managing quality across multiple repositories introduces challenges:

- **Configuration Drift**: Each repo may have different quality standards
- **Tool Version Skew**: Different projects using different tool versions
- **Inconsistent Enforcement**: Some repos enforce quality, others don't
- **Reporting Fragmentation**: No unified view across repositories
- **Maintenance Overhead**: Updates must be replicated across repos

### Solutions

gz-quality addresses these with:

- ✅ Centralized configuration via shared configs
- ✅ Consistent tool versions via pinned releases
- ✅ Reusable CI/CD templates
- ✅ Aggregated reporting across repos
- ✅ Automated updates via dependabot/renovate

---

## Repository Types

### 1. Monorepo

**Structure**: Single repository containing multiple projects/services

```
monorepo/
├── .gzquality.yml          # Root config
├── services/
│   ├── api/
│   │   ├── .gzquality.yml  # Override for API
│   │   └── main.go
│   ├── worker/
│   │   ├── .gzquality.yml  # Override for worker
│   │   └── main.py
│   └── frontend/
│       ├── .gzquality.yml  # Override for frontend
│       └── index.ts
└── shared/
    └── lib.go
```

**Benefits**:
- Single CI/CD pipeline
- Consistent tooling
- Atomic cross-project changes

**Challenges**:
- Longer CI times
- Different language requirements per service

---

### 2. Polyrepo

**Structure**: Multiple independent repositories

```
org/
├── backend-api/        # Go service
│   └── .gzquality.yml
├── backend-worker/     # Python service
│   └── .gzquality.yml
├── frontend-web/       # TypeScript app
│   └── .gzquality.yml
└── mobile-app/         # React Native
    └── .gzquality.yml
```

**Benefits**:
- Independent deployment
- Focused CI/CD
- Clear ownership

**Challenges**:
- Configuration duplication
- Version drift
- Cross-repo changes are complex

---

### 3. Hybrid (Monorepo + Libraries)

**Structure**: Core monorepo + separate library repos

```
org/
├── platform/           # Monorepo (core services)
│   ├── services/
│   └── .gzquality.yml
├── lib-auth/           # Shared library
│   └── .gzquality.yml
└── lib-logging/        # Shared library
    └── .gzquality.yml
```

**Benefits**:
- Core services together
- Libraries independently versioned
- Flexible architecture

**Challenges**:
- More complex CI/CD
- Dependency management

---

## Workflow Patterns

### Pattern 1: Shared Configuration Repository

Centralize quality configuration in a dedicated repo:

```
org/config-quality/
├── .gzquality.base.yml         # Base config
├── .gzquality.backend.yml      # Backend overrides
├── .gzquality.frontend.yml     # Frontend overrides
└── .github/
    └── workflows/
        └── quality-check.yml   # Reusable workflow
```

**Usage in project repos**:

```yaml
# backend-api/.gzquality.yml
extends: https://raw.githubusercontent.com/org/config-quality/main/.gzquality.backend.yml

# Project-specific overrides
tools:
  golangci-lint:
    timeout: "5m"  # Override for large project
```

**Benefits**:
- ✅ Single source of truth
- ✅ Easy to update all repos
- ✅ Version controlled configuration
- ✅ Rollback support

---

### Pattern 2: Per-Repository Configuration

Each repo maintains its own configuration:

```
backend-api/.gzquality.yml
backend-worker/.gzquality.yml
frontend-web/.gzquality.yml
```

**Benefits**:
- ✅ Maximum flexibility
- ✅ No external dependencies
- ✅ Repository-specific tuning

**Challenges**:
- ❌ Configuration drift
- ❌ Hard to enforce consistency
- ❌ Updates require multiple PRs

**When to Use**:
- Projects have vastly different requirements
- Teams have strong autonomy
- Flexibility > consistency

---

### Pattern 3: Template Repository

Use GitHub template repos for new projects:

```
org/template-go-service/
├── .gzquality.yml
├── .github/
│   └── workflows/
│       └── quality.yml
├── .pre-commit-config.yaml
└── README.md
```

**Benefits**:
- ✅ New projects start with quality checks
- ✅ Consistent baseline configuration
- ✅ Pre-configured CI/CD

**Challenges**:
- ❌ Divergence after creation
- ❌ Updates not automatic

**Enhancement**: Combine with Pattern 1 (shared config) for updates

---

### Pattern 4: Git Submodules

Share configuration via git submodules:

```
backend-api/
├── .git/
├── config/              # Git submodule
│   └── .gzquality.yml
└── main.go

# .gzquality.yml symlink → config/.gzquality.yml
```

**Benefits**:
- ✅ Version-controlled configuration
- ✅ Easy to update across repos (`git submodule update`)
- ✅ Explicit version pinning

**Challenges**:
- ❌ Git submodule complexity
- ❌ Requires manual updates
- ❌ Merge conflicts possible

---

## Configuration Management

### Base Configuration

Create a minimal, extensible base configuration:

```yaml
# .gzquality.base.yml
version: 1

# Common excludes
exclude:
  - "vendor/**"
  - "node_modules/**"
  - "**/*_gen.go"
  - "**/*.pb.go"
  - "**/dist/**"
  - "**/build/**"

# Conservative timeout
timeout: "10m"

# Fail on errors, warn on warnings
strict: false

# Enable auto-fix for formats
autofix: true
```

### Language-Specific Overlays

Extend base config with language-specific settings:

```yaml
# .gzquality.backend.yml
extends: .gzquality.base.yml

tools:
  golangci-lint:
    enabled: true
    timeout: "5m"
    args:
      - "--enable=gofmt,goimports,govet"
      - "--disable=unused"

  gofumpt:
    enabled: true

  goimports:
    enabled: true
```

```yaml
# .gzquality.frontend.yml
extends: .gzquality.base.yml

tools:
  prettier:
    enabled: true
    args:
      - "--write"
      - "--print-width=100"

  eslint:
    enabled: true
    args:
      - "--fix"
      - "--max-warnings=0"

  tsc:
    enabled: true
```

### Project-Specific Overrides

Override in individual repositories:

```yaml
# backend-api/.gzquality.yml
extends: https://raw.githubusercontent.com/org/config-quality/main/.gzquality.backend.yml

# Project needs longer timeout
timeout: "15m"

# Project-specific excludes
exclude:
  - "internal/generated/**"
  - "pkg/proto/**"

# Disable specific tool
tools:
  golangci-lint:
    timeout: "10m"  # Larger project

  goimports:
    enabled: false  # Using gofumpt only
```

---

## CI/CD Strategies

### Strategy 1: Reusable Workflow (GitHub Actions)

Create a reusable workflow in a central repo:

```yaml
# org/.github/workflows/quality-check.yml
name: Reusable Quality Check

on:
  workflow_call:
    inputs:
      go-version:
        description: 'Go version'
        default: '1.24'
        type: string
      gz-quality-version:
        description: 'gz-quality version'
        default: 'v0.1.1'
        type: string
      report-format:
        description: 'Report format'
        default: 'json'
        type: string

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ inputs.go-version }}

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@${{ inputs.gz-quality-version }}

      - name: Run quality check
        run: |
          gz-quality check \
            --since origin/${{ github.base_ref || 'main' }} \
            --report ${{ inputs.report-format }} \
            --output quality-report.${{ inputs.report-format }}

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: quality-report
          path: quality-report.${{ inputs.report-format }}
```

**Usage in project repos**:

```yaml
# backend-api/.github/workflows/quality.yml
name: Quality Check

on:
  pull_request:
    branches: [main]

jobs:
  quality:
    uses: org/.github/.github/workflows/quality-check.yml@main
    with:
      go-version: '1.24'
      gz-quality-version: 'v0.1.1'
```

**Benefits**:
- ✅ DRY: Single workflow definition
- ✅ Easy updates (update once, affects all repos)
- ✅ Consistent behavior across repos
- ✅ Centralized maintenance

---

### Strategy 2: Composite Action

Package quality check as a composite action:

```yaml
# org/actions/quality-check/action.yml
name: 'Quality Check'
description: 'Run gz-quality checks'

inputs:
  gz-quality-version:
    description: 'gz-quality version'
    default: 'v0.1.1'
  mode:
    description: 'Check mode'
    default: 'check'
  report-format:
    description: 'Report format'
    default: 'json'

runs:
  using: 'composite'
  steps:
    - name: Install gz-quality
      shell: bash
      run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@${{ inputs.gz-quality-version }}

    - name: Run quality check
      shell: bash
      run: |
        gz-quality ${{ inputs.mode }} \
          --report ${{ inputs.report-format }} \
          --output quality-report.${{ inputs.report-format }}
```

**Usage**:

```yaml
# backend-api/.github/workflows/ci.yml
- uses: org/actions/quality-check@v1
  with:
    gz-quality-version: 'v0.1.1'
    mode: 'check'
```

---

### Strategy 3: Shared Scripts

Use shell scripts for CI-agnostic implementation:

```bash
#!/bin/bash
# org/scripts/run-quality-check.sh

set -e

VERSION=${GZ_QUALITY_VERSION:-v0.1.1}
MODE=${GZ_QUALITY_MODE:-check}
REPORT=${GZ_QUALITY_REPORT:-json}

echo "Installing gz-quality $VERSION..."
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@$VERSION

echo "Running quality $MODE..."
gz-quality $MODE \
  --report $REPORT \
  --output quality-report.$REPORT

echo "Quality check completed!"
```

**Usage in any CI**:

```yaml
# GitHub Actions
- run: bash <(curl -fsSL https://raw.githubusercontent.com/org/scripts/main/run-quality-check.sh)

# GitLab CI
script:
  - curl -fsSL https://raw.githubusercontent.com/org/scripts/main/run-quality-check.sh | bash

# CircleCI
- run: curl -fsSL https://raw.githubusercontent.com/org/scripts/main/run-quality-check.sh | bash
```

---

## Tool Installation

### Centralized Tool Version Management

Use a version manifest:

```yaml
# org/config-quality/tool-versions.yml
gz-quality: v0.1.1
golangci-lint: v1.64.8
ruff: v0.1.9
prettier: v3.1.0
eslint: v8.56.0
```

**Installation script**:

```bash
#!/bin/bash
# install-tools.sh

set -e

# Fetch version manifest
VERSIONS_URL="https://raw.githubusercontent.com/org/config-quality/main/tool-versions.yml"
VERSIONS=$(curl -fsSL $VERSIONS_URL)

# Parse and install
GZ_QUALITY_VERSION=$(echo "$VERSIONS" | grep 'gz-quality:' | awk '{print $2}')
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@$GZ_QUALITY_VERSION

# Install other tools as needed
```

---

### Docker Image Strategy

Build a custom Docker image with all tools:

```dockerfile
# org/docker/quality-tools/Dockerfile
FROM golang:1.24-alpine

# Install gz-quality
ARG GZ_QUALITY_VERSION=v0.1.1
RUN go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@${GZ_QUALITY_VERSION}

# Install Go tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 && \
    go install mvdan.cc/gofumpt@latest

# Install Python tools
RUN apk add --no-cache python3 py3-pip && \
    pip3 install black ruff pylint

# Install Node tools
RUN apk add --no-cache nodejs npm && \
    npm install -g prettier eslint

WORKDIR /workspace
ENTRYPOINT ["gz-quality"]
```

**Usage**:

```yaml
# .github/workflows/quality.yml
jobs:
  quality:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/org/quality-tools:v0.1.1
    steps:
      - uses: actions/checkout@v4
      - run: gz-quality check
```

**Benefits**:
- ✅ Consistent tool versions across all repos
- ✅ Fast CI (pre-installed tools)
- ✅ Easy to update (rebuild image)
- ✅ Reproducible builds

---

## Reporting and Aggregation

### Individual Repository Reports

Each repo generates its own report:

```bash
# backend-api
gz-quality check --report json --output reports/backend-api-quality.json

# backend-worker
gz-quality check --report json --output reports/backend-worker-quality.json

# frontend-web
gz-quality check --report json --output reports/frontend-web-quality.json
```

---

### Centralized Dashboard

Aggregate reports in a central location:

```yaml
# org/.github/workflows/aggregate-quality.yml
name: Aggregate Quality Reports

on:
  schedule:
    - cron: '0 0 * * *'  # Daily
  workflow_dispatch:

jobs:
  aggregate:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout dashboard repo
        uses: actions/checkout@v4
        with:
          repository: org/quality-dashboard

      - name: Fetch reports from all repos
        run: |
          # Download latest reports
          gh run download --repo org/backend-api --name quality-report
          gh run download --repo org/backend-worker --name quality-report
          gh run download --repo org/frontend-web --name quality-report

      - name: Aggregate reports
        run: |
          python scripts/aggregate_reports.py \
            --input reports/ \
            --output dashboard/data.json

      - name: Deploy dashboard
        run: |
          npm run build
          npm run deploy
```

---

### Report Aggregation Script

```python
#!/usr/bin/env python3
# scripts/aggregate_reports.py

import json
import glob
from pathlib import Path

def aggregate_reports(input_dir, output_file):
    reports = []

    for report_file in glob.glob(f"{input_dir}/*-quality.json"):
        with open(report_file) as f:
            data = json.load(f)
            repo_name = Path(report_file).stem.replace('-quality', '')

            reports.append({
                "repository": repo_name,
                "timestamp": data.get("timestamp"),
                "summary": data.get("summary"),
                "results": data.get("results", [])
            })

    # Calculate organization-wide metrics
    total_files = sum(r["summary"]["total_files"] for r in reports)
    total_issues = sum(r["summary"]["total_issues"] for r in reports)

    aggregated = {
        "timestamp": datetime.now().isoformat(),
        "organization_summary": {
            "total_repositories": len(reports),
            "total_files_checked": total_files,
            "total_issues_found": total_issues,
            "average_issues_per_repo": total_issues / len(reports) if reports else 0
        },
        "repositories": reports
    }

    with open(output_file, 'w') as f:
        json.dump(aggregated, f, indent=2)

if __name__ == '__main__':
    import sys
    aggregate_reports(sys.argv[1], sys.argv[2])
```

---

## Real-World Examples

### Example 1: Microservices Organization

**Structure**:
- 20 Go microservices
- 5 Python data pipelines
- 3 TypeScript frontends

**Solution**:

```
org/
├── config-quality/                 # Central config repo
│   ├── .gzquality.base.yml
│   ├── .gzquality.backend.yml
│   ├── .gzquality.frontend.yml
│   └── .github/workflows/
│       └── quality-check.yml      # Reusable workflow
│
├── service-api-1/                  # Go service
│   ├── .gzquality.yml             # extends: .backend.yml
│   └── .github/workflows/
│       └── ci.yml                 # uses: quality-check.yml
│
├── service-api-2/                  # Go service
│   ├── .gzquality.yml             # extends: .backend.yml
│   └── .github/workflows/
│       └── ci.yml                 # uses: quality-check.yml
│
└── quality-dashboard/              # Dashboard repo
    ├── scripts/aggregate_reports.py
    └── public/index.html
```

**Benefits**:
- Update once in `config-quality`, affects all 28 repos
- Consistent quality standards organization-wide
- Central dashboard for monitoring
- Automated daily quality reports

---

### Example 2: Platform + Libraries

**Structure**:
- 1 large monorepo (platform)
- 10 shared library repos

**Solution**:

```yaml
# platform/.gzquality.yml (monorepo)
version: 1

# Check only changed services
exclude:
  - "services/*/vendor/**"

tools:
  golangci-lint:
    enabled: true

  gofumpt:
    enabled: true

# Per-service overrides
overrides:
  - path: "services/api/**"
    tools:
      golangci-lint:
        timeout: "10m"  # Largest service

  - path: "services/worker/**"
    tools:
      pylint:
        enabled: true
```

```yaml
# lib-auth/.gzquality.yml (library)
extends: https://raw.githubusercontent.com/org/config-quality/main/.gzquality.library.yml

# Library-specific: stricter checks
strict: true

tools:
  golangci-lint:
    args:
      - "--enable=gocyclo,dupl,gocognit"  # More linters
```

**Workflow**:

```yaml
# platform/.github/workflows/quality.yml
on:
  pull_request:
    paths:
      - 'services/**'

jobs:
  detect-changes:
    outputs:
      services: ${{ steps.changes.outputs.services }}
    steps:
      - uses: dorny/paths-filter@v2
        id: changes
        with:
          filters: |
            services:
              - 'services/**'

  quality-check:
    needs: detect-changes
    if: needs.detect-changes.outputs.services == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1

      - name: Check changed services only
        run: gz-quality check --since origin/${{ github.base_ref }}
```

**Benefits**:
- Monorepo: Check only changed services (fast)
- Libraries: Stricter checks (higher quality bar)
- Independent CI pipelines
- Reusable configuration

---

### Example 3: Multi-Region Teams

**Structure**:
- US team: 5 repos (Go services)
- EU team: 5 repos (Python services)
- APAC team: 3 repos (TypeScript frontends)

**Solution**: Regional config repositories

```
org/
├── config-quality-us/          # US team config
│   └── .gzquality.yml
├── config-quality-eu/          # EU team config
│   └── .gzquality.yml
├── config-quality-apac/        # APAC team config
│   └── .gzquality.yml
└── config-quality-global/      # Shared base
    └── .gzquality.base.yml
```

```yaml
# config-quality-us/.gzquality.yml
extends: https://raw.githubusercontent.com/org/config-quality-global/main/.gzquality.base.yml

# US team: Go-focused
tools:
  golangci-lint:
    enabled: true
    timeout: "5m"

  gofumpt:
    enabled: true
```

```yaml
# config-quality-eu/.gzquality.yml
extends: https://raw.githubusercontent.com/org/config-quality-global/main/.gzquality.base.yml

# EU team: Python-focused
tools:
  ruff:
    enabled: true

  black:
    enabled: true

  pylint:
    enabled: true
```

**Benefits**:
- Team autonomy with shared baseline
- Regional language preferences
- Global consistency where needed
- Easy to add new teams

---

## Best Practices

### 1. Version Everything

Pin all versions explicitly:

```yaml
# .github/workflows/quality.yml
- name: Install gz-quality
  run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
  # ❌ DON'T: @latest (unpredictable)
  # ✅ DO: @v0.1.1 (reproducible)
```

---

### 2. Automate Updates

Use Dependabot or Renovate:

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    groups:
      quality-tools:
        patterns:
          - "github.com/Gizzahub/gzh-cli-quality"
```

---

### 3. Monitor Drift

Regularly audit configuration drift:

```bash
#!/bin/bash
# scripts/audit-config-drift.sh

REPOS=("backend-api" "backend-worker" "frontend-web")

for repo in "${REPOS[@]}"; do
    echo "Checking $repo..."

    # Download .gzquality.yml
    curl -fsSL "https://raw.githubusercontent.com/org/$repo/main/.gzquality.yml" > "/tmp/$repo.yml"

    # Compare with base
    diff -u .gzquality.base.yml "/tmp/$repo.yml" || true
done
```

---

### 4. Gradual Rollout

When updating configurations:

```
Week 1: Update config-quality repo (PR + review)
Week 2: Pilot in 2-3 repos
Week 3: Roll out to 25% of repos
Week 4: Roll out to remaining repos
```

---

### 5. Escape Hatches

Allow repository-specific overrides when needed:

```yaml
# backend-api/.gzquality.yml
extends: https://raw.githubusercontent.com/org/config-quality/main/.gzquality.backend.yml

# Temporary: Disable problematic linter
tools:
  golangci-lint:
    args:
      - "--disable=errcheck"  # FIXME: Re-enable after addressing errors

# Document why override is needed
# Issue: https://github.com/org/backend-api/issues/123
```

---

### 6. Measure and Report

Track quality metrics organization-wide:

```yaml
# Dashboard metrics
- Total repositories: 28
- Repositories with gz-quality: 28 (100%)
- Average issues per repo: 12.3
- Trend: -15% vs last month ✅
```

---

### 7. Documentation

Maintain a central documentation hub:

```
org/quality-handbook/
├── README.md               # Overview
├── getting-started.md      # Onboarding
├── configuration.md        # Config reference
├── workflows.md            # CI/CD patterns
├── troubleshooting.md      # Common issues
└── examples/               # Real examples
```

---

### 8. Regular Reviews

Schedule quarterly reviews:

- Review tool versions (upgrade strategy)
- Assess configuration drift
- Gather team feedback
- Update documentation
- Celebrate improvements

---

## Related Documentation

- [CI Integration Guide](./CI_INTEGRATION.md) - CI/CD integration patterns
- [Pre-commit Hooks Guide](./PRE_COMMIT_HOOKS.md) - Local quality checks
- [Configuration Reference](../developer/API.md) - Complete configuration options
- [Usage Examples](../user/02-examples.md) - Detailed usage examples

---

**Last Updated**: 2025-11-27
**gz-quality Version**: v0.1.1

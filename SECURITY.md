# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

The gzh-cli-quality team takes security bugs seriously. We appreciate your efforts to responsibly disclose your findings.

### Where to Report

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

1. **Email**: Send details to security@gizzahub.com (preferred)
2. **GitHub Security Advisory**: Use the "Security" tab to privately report a vulnerability

### What to Include

To help us triage and fix the issue quickly, please include:

- Type of issue (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### Response Timeline

- **Initial Response**: Within 48 hours
- **Triage**: Within 7 days
- **Fix Development**: Depends on severity and complexity
- **Public Disclosure**: Coordinated with reporter after fix is released

### Disclosure Policy

- Security issues are handled with highest priority
- Fixes are developed privately and tested
- CVE IDs are requested for confirmed vulnerabilities
- Public disclosure happens after patch is available
- Credit is given to reporters (unless anonymity is requested)

## Security Best Practices for Users

When using gzh-cli-quality:

### 1. Keep Updated

Always use the latest version:

```bash
go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@latest
```

### 2. Review Tool Output

- Carefully review changes made by `--fix` flag before committing
- Use `--dry-run` to preview execution plan
- Verify tool versions: `gzq version`

### 3. Configuration Security

- Don't commit sensitive data in `.gzquality.yml`
- Use `.gitignore` to exclude sensitive configs
- Review tool-specific security settings

### 4. CI/CD Integration

- Run gzq in read-only mode in CI: `gzq check`
- Don't use `--fix` in CI without manual review
- Limit permissions for CI service accounts

### 5. Tool Execution

gzh-cli-quality executes external quality tools (golangci-lint, ruff, etc.):

- Only install tools from trusted sources
- Keep quality tools updated
- Review tool configurations for security implications
- Be cautious with `extra-args` flags

## Known Security Considerations

### Command Execution

gzh-cli-quality executes external commands. While we sanitize inputs:

- Tool names are validated against a whitelist
- File paths are validated
- No shell interpretation of user input
- All commands use `exec.Command`, not shell execution

### File System Access

- Tools only access files within project directory
- No writes outside current working directory
- Configuration files are read-only

### Network Access

- No network access required for core functionality
- Tool installation (`gzq install`) may download packages
- Only uses official package managers (go install, pip, npm, cargo)

## Security Updates

Security updates are published as:

1. New releases on GitHub
2. Security advisories on GitHub Security
3. Announcements in project README
4. CVE database entries (for severe issues)

## Bug Bounty Program

We currently do not have a bug bounty program. However, we deeply appreciate security researchers who report vulnerabilities responsibly and will:

- Publicly acknowledge your contribution (with permission)
- Keep you informed throughout the fix process
- Work with you on disclosure timing

## Questions?

For security-related questions that are not vulnerabilities:

- Open a discussion: https://github.com/Gizzahub/gzh-cli-quality/discussions
- Email: security@gizzahub.com

---

**Last Updated**: 2025-11-27

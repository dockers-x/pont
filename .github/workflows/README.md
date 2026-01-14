# CI/CD Workflows

## Overview

This directory contains GitHub Actions workflows for Pont's CI/CD pipeline.

## Workflows

### ci.yml - Main CI/CD Pipeline

Comprehensive workflow that handles testing, security scanning, Docker builds, and releases.

#### Triggers

- **Push to main**: Runs tests, security scans, and builds Docker images
- **Pull requests**: Runs tests and security scans only
- **Tags (v*)**: Runs full release process including binary builds

#### Jobs

##### 1. Test
- Runs on every push and PR
- Sets up Go 1.24
- Downloads and verifies dependencies
- Runs `go vet` for static analysis
- Runs tests with race detection and coverage
- Builds the application

##### 2. Security
- Runs Gosec security scanner
- Runs govulncheck for vulnerability detection
- Fails if critical vulnerabilities found

##### 3. Docker Build (Push only)
- Multi-architecture builds (amd64, arm64)
- Uses Docker Buildx for cross-platform builds
- Pushes to GitHub Container Registry (ghcr.io)
- Implements layer caching for faster builds
- Builds by digest for manifest merging

##### 4. Docker Merge (Push only)
- Merges multi-arch images into single manifest
- Tags images appropriately:
  - Branch name for branch pushes
  - Semantic versions for tags (v1.2.3, v1.2, v1)
  - Git SHA for traceability

##### 5. Release (Tags only)
- Builds binaries for multiple platforms:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- Creates GitHub release with binaries
- Generates release notes automatically
- Injects version information into binaries

## Setup Requirements

### Secrets

No additional secrets required! The workflow uses:
- `GITHUB_TOKEN` (automatically provided)

### Optional: Docker Hub

To also push to Docker Hub, add these secrets:
- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

Then update the workflow to add Docker Hub login.

### Permissions

The workflow requires:
- `contents: write` - For creating releases
- `packages: write` - For pushing to GHCR

These are configured in the workflow file.

## Usage

### Running Tests Locally

```bash
# Run tests
go test -v -race ./...

# Run security scan
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...

# Run vulnerability check
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### Building Docker Images Locally

```bash
# Build for current platform
docker build -t supertunnel:local .

# Build multi-arch (requires buildx)
docker buildx build --platform linux/amd64,linux/arm64 -t supertunnel:local .
```

### Creating a Release

1. Tag the commit:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. The workflow will automatically:
   - Run all tests and security scans
   - Build binaries for all platforms
   - Create a GitHub release
   - Build and push Docker images with version tags

### Docker Image Tags

Images are available at `ghcr.io/<owner>/supertunnel`:

- `main` - Latest main branch build
- `v1.2.3` - Specific version
- `v1.2` - Minor version (latest patch)
- `v1` - Major version (latest minor)
- `sha-<commit>` - Specific commit

### Pulling Images

```bash
# Latest main branch
docker pull ghcr.io/<owner>/supertunnel:main

# Specific version
docker pull ghcr.io/<owner>/supertunnel:v1.0.0

# Specific architecture
docker pull --platform linux/arm64 ghcr.io/<owner>/supertunnel:v1.0.0
```

## Caching

The workflow implements several caching strategies:

1. **Go modules**: Cached by `actions/setup-go`
2. **Docker layers**: Cached using GitHub Actions cache
3. **Build artifacts**: Shared between jobs using artifacts

## Troubleshooting

### Build Failures

Check the workflow logs for specific errors:
- Test failures: Review test output
- Security issues: Check Gosec/govulncheck reports
- Docker build issues: Verify Dockerfile syntax

### Release Issues

Ensure:
- Tag follows `v*` pattern (e.g., v1.0.0)
- Repository has releases enabled
- Workflow has proper permissions

### Docker Push Failures

Verify:
- GITHUB_TOKEN has packages:write permission
- Repository allows package publishing
- Image name matches repository name

## Best Practices

1. **Always run tests locally** before pushing
2. **Use semantic versioning** for releases (v1.2.3)
3. **Review security scan results** in PR checks
4. **Test Docker images** before tagging releases
5. **Keep dependencies updated** regularly

## Maintenance

### Updating Go Version

Update `GO_VERSION` in the workflow:

```yaml
env:
  GO_VERSION: '1.24'
```

### Adding New Platforms

Add to the release matrix:

```yaml
matrix:
  include:
    - goos: freebsd
      goarch: amd64
```

### Customizing Docker Tags

Modify the `docker/metadata-action` tags section:

```yaml
tags: |
  type=ref,event=branch
  type=semver,pattern={{version}}
  type=raw,value=latest,enable={{is_default_branch}}
```

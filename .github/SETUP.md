# CI/CD Setup Guide

## Initial Setup

### 1. Generate go.sum

Before the CI/CD pipeline can run, generate the `go.sum` file:

```bash
go mod download
go mod tidy
```

### 2. Enable GitHub Actions

1. Go to repository Settings > Actions > General
2. Enable "Allow all actions and reusable workflows"
3. Under "Workflow permissions", select "Read and write permissions"
4. Check "Allow GitHub Actions to create and approve pull requests"

### 3. Enable GitHub Packages

1. Go to repository Settings > Actions > General
2. Scroll to "Workflow permissions"
3. Ensure "Read and write permissions" is selected

### 4. Test the Workflow

Create a test commit:

```bash
git add .
git commit -m "Add CI/CD workflow"
git push origin main
```

Check the Actions tab to see the workflow run.

### 5. Create First Release

```bash
# Tag the release
git tag -a v0.1.0 -m "Initial release"
git push origin v0.1.0
```

The workflow will:
- Build binaries for all platforms
- Create a GitHub release
- Build and push Docker images

## Verification

### Check Docker Images

```bash
# View packages in GitHub
# Go to: https://github.com/<owner>/<repo>/pkgs/container/supertunnel

# Pull and test
docker pull ghcr.io/<owner>/supertunnel:v0.1.0
docker run --rm ghcr.io/<owner>/supertunnel:v0.1.0 --version
```

### Check Release Artifacts

1. Go to Releases page
2. Download a binary for your platform
3. Test it locally

## Optional Enhancements

### Add Docker Hub Publishing

1. Create Docker Hub access token
2. Add repository secrets:
   - `DOCKERHUB_USERNAME`
   - `DOCKERHUB_TOKEN`

3. Update workflow to add Docker Hub login:

```yaml
- name: Login to Docker Hub
  uses: docker/login-action@v3
  with:
    username: ${{ secrets.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}
```

4. Update `REGISTRY_IMAGE` to include Docker Hub:

```yaml
env:
  REGISTRY_IMAGE: |
    ghcr.io/${{ github.repository }}
    docker.io/<dockerhub-username>/supertunnel
```

### Add Status Badges

Add to README.md:

```markdown
[![CI/CD](https://github.com/<owner>/<repo>/actions/workflows/ci.yml/badge.svg)](https://github.com/<owner>/<repo>/actions/workflows/ci.yml)
[![Docker Image](https://ghcr-badge.egpl.dev/<owner>/<repo>/latest_tag?trim=major&label=latest)](https://github.com/<owner>/<repo>/pkgs/container/supertunnel)
```

### Add Dependabot

Create `.github/dependabot.yml`:

```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
```

## Troubleshooting

### Workflow Not Running

- Check Actions are enabled in repository settings
- Verify workflow file syntax with `yamllint`
- Check branch protection rules

### Docker Push Fails

- Verify packages:write permission
- Check if package already exists with different visibility
- Ensure repository name matches image name

### Release Creation Fails

- Verify contents:write permission
- Check if tag already exists
- Ensure tag follows v* pattern

### Build Fails

- Run `go mod tidy` to fix dependency issues
- Check Go version compatibility
- Review build logs for specific errors

# CI/CD Documentation

This document describes the continuous integration and deployment workflow for git-kanban.

## Overview

The GitHub Actions workflow (`build-and-release.yml`) automatically builds, tests, and packages git-kanban on:
- **Pull Requests** to main/master branches
- **Pushes** to main/master branches  
- **Tag pushes** (e.g., `v1.0.0`)

## Workflow Jobs

### 1. Build Job

Builds the Go web server binary for multiple platforms:

- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64

Each build:
- Uses Go 1.21
- Compiles with `-ldflags="-s -w"` for smaller binaries
- Disables CGO for static linking
- Uploads binaries as workflow artifacts (30-day retention)

### 2. Test Job

Runs all tests to verify functionality:

1. **Shell Integration Tests**: `./tests/test-lanes.sh`
   - Creates temporary git repo
   - Tests owner inference with "kanban:" prefix filtering
   - Validates TSV output format

2. **Go Unit Tests**: `go test -v ./...`
   - Tests TSV parsing
   - Tests owner merging logic
   - Tests board I/O functions

3. **Smoke Tests**: Basic verification of shell script
   - `--help` flag
   - `--lanes` flag

### 3. Package Job

Creates release packages (runs after build job):

**For All Events (PR/Push/Tag):**
- Downloads all build artifacts
- Creates archives:
  - `git-kanban-shell.tar.gz` - Shell script + install scripts + README
  - `git-kanban-web-all-platforms.tar.gz` - All web server binaries
  - Platform-specific archives (e.g., `git-kanban-web-linux-amd64.tar.gz`)
- Uploads as workflow artifacts (30-day retention)

**For Tag Pushes Only:**
- Generates SHA256 checksums
- Creates GitHub Release with:
  - All archives
  - Checksums file
  - Auto-generated release notes

## Usage

### On Pull Requests

When you open a PR to main/master:
1. Build job compiles binaries for all platforms
2. Test job runs all tests
3. Package job creates archives
4. Artifacts available for download from the Actions tab

**Viewing Artifacts:**
- Go to PR → Checks → Build and Release workflow
- Click on completed job
- Scroll to "Artifacts" section
- Download `release-packages` for all archives

### On Merge to Main

When PR is merged:
1. Same build, test, and package steps run
2. Artifacts available for 30 days
3. Latest main build is always accessible

### Creating a Release

To create an official release with GitHub Release:

```bash
# Create and push a tag
git tag v1.0.0
git push origin v1.0.0
```

The workflow will:
1. Build all platform binaries
2. Run all tests
3. Create release packages
4. Generate checksums
5. Create a GitHub Release with all assets

**Release Assets:**
- `git-kanban-shell.tar.gz` - Shell script package
- `git-kanban-web-linux-amd64.tar.gz` - Linux x64 web server
- `git-kanban-web-linux-arm64.tar.gz` - Linux ARM64 web server
- `git-kanban-web-darwin-amd64.tar.gz` - macOS Intel web server
- `git-kanban-web-darwin-arm64.tar.gz` - macOS Apple Silicon web server
- `git-kanban-web-windows-amd64.tar.gz` - Windows x64 web server
- `git-kanban-web-all-platforms.tar.gz` - All binaries
- `SHA256SUMS.txt` - Checksums for verification

## Package Contents

### git-kanban-shell.tar.gz

```
git-kanban-shell/
├── git-kanban              # Main shell script (TUI + --lanes)
├── install-git-kanban.sh   # Linux/macOS installer
├── install-git-kanban.ps1  # Windows installer
└── README.md               # Documentation
```

**Installation:**
```bash
tar -xzf git-kanban-shell.tar.gz
cd git-kanban-shell
./install-git-kanban.sh
```

### git-kanban-web-<platform>.tar.gz

```
git-kanban-web-<platform>/
├── git-kanban-web-<platform>  # Web server binary
└── README.md                   # Documentation
```

**Usage:**
```bash
tar -xzf git-kanban-web-linux-amd64.tar.gz
cd git-kanban-web-linux-amd64
./git-kanban-web-linux-amd64
# Open http://localhost:8080/static/index.html
```

## Build Configuration

### Go Build Settings

- **Go Version**: 1.21
- **Build Flags**: `-ldflags="-s -w"`
  - `-s`: Strip symbol table
  - `-w`: Strip DWARF debug info
  - Result: ~50% smaller binaries
- **CGO**: Disabled for static linking
- **Platforms**: Cross-compiled using GOOS/GOARCH

### Artifact Retention

- **PR/Push Artifacts**: 30 days
- **GitHub Releases**: Permanent

## Troubleshooting

### Build Failures

**Problem**: Build job fails
- Check Go version compatibility
- Verify all dependencies in `go.mod`
- Review build logs in Actions tab

**Problem**: Tests fail
- Check if changes broke existing functionality
- Run tests locally: `./tests/test-lanes.sh` and `go test ./...`
- Review test output in Actions logs

### Missing Artifacts

**Problem**: No artifacts on PR
- Ensure workflow file is in `.github/workflows/`
- Check workflow syntax (YAML validation)
- Verify PR targets main/master branch

**Problem**: Release not created on tag
- Ensure tag matches pattern `v*` (e.g., `v1.0.0`)
- Check `GITHUB_TOKEN` has release permissions
- Review Actions logs for release job

## Local Testing

To test the workflow locally before pushing:

```bash
# Install act (GitHub Actions local runner)
# https://github.com/nektos/act

# Test build job
act pull_request -j build

# Test package job (requires build artifacts)
act pull_request -j package

# Test release creation (requires tag)
act -e <(echo '{"ref":"refs/tags/v1.0.0"}') push
```

## Maintenance

### Updating Go Version

Edit `.github/workflows/build-and-release.yml`:

```yaml
- name: Set up Go
  uses: actions/setup-go@v4
  with:
    go-version: '1.22'  # Update version
```

### Adding New Platforms

Add to the build matrix:

```yaml
- goos: freebsd
  goarch: amd64
  output: git-kanban-web-freebsd-amd64
```

### Changing Artifact Retention

Modify `retention-days`:

```yaml
- name: Upload build artifact
  uses: actions/upload-artifact@v3
  with:
    retention-days: 90  # Change from 30
```

## Security

- **GITHUB_TOKEN**: Automatically provided by GitHub Actions
- **No secrets required**: All builds are from public code
- **Checksums**: SHA256 sums provided for release verification

## Performance

Typical run times:
- **Build job**: 2-3 minutes (parallel builds)
- **Test job**: 1-2 minutes
- **Package job**: 1-2 minutes
- **Total**: ~5-7 minutes

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go Cross Compilation](https://go.dev/doc/install/source#environment)
- [softprops/action-gh-release](https://github.com/softprops/action-gh-release)

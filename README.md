# specview

**A local web interface for reading your [OpenSpec](https://openspec.dev) projects.**

`specview` starts from your terminal and opens your project in the browser. Browse specifications, active changes, and archives, read artifacts, and track task progress. The interface refreshes when files change and suggests the next OpenSpec command to copy into your agent or terminal.

The application is read-only: your files remain the single source of truth. The frontend is embedded in a standalone Go binary that works offline.

## Installation

Download the binary for your system from **[GitHub Releases](../../releases)**.

| System | Architecture | Binary |
| --- | --- | --- |
| macOS | Apple Silicon | `specview-darwin-arm64` |
| macOS | Intel | `specview-darwin-amd64` |
| Linux | x86-64 | `specview-linux-amd64` |
| Linux | ARM64 | `specview-linux-arm64` |
| Windows | x86-64 | `specview-windows-amd64.exe` |
| Windows | ARM64 | `specview-windows-arm64.exe` |

You only need a browser and a project containing an `openspec/` directory. Go, Node.js, and the OpenSpec CLI are not required to run `specview`.

### macOS and Linux

After downloading, install the binary as `specview`. For example, on an Apple Silicon Mac:

```sh
mkdir -p "$HOME/.local/bin"
install -m 755 "$HOME/Downloads/specview-darwin-arm64" "$HOME/.local/bin/specview"
export PATH="$HOME/.local/bin:$PATH"
specview --version
```

On Linux or an Intel Mac, replace the filename with the appropriate one from the table. Adjust the download path if needed.

To make the command available in future terminal sessions, add this line to your shell configuration (`~/.zshrc` for zsh or `~/.bashrc` for bash), if necessary:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

### Windows

In PowerShell, install the binary as `specview.exe`:

```powershell
$installDir = Join-Path $env:LOCALAPPDATA 'Programs\specview'
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Copy-Item "$HOME\Downloads\specview-windows-amd64.exe" (Join-Path $installDir 'specview.exe')
& "$installDir\specview.exe" --version
```

On an ARM64 machine, use `specview-windows-arm64.exe`. Add the installation directory to your user `Path` in Windows environment variables, then open a new terminal to use the `specview` command.

### Updating

Stop `specview` with `Ctrl+C`, download the new binary from Releases, and replace the installed file by repeating the steps above. Your OpenSpec projects do not require a migration.

## Usage

From an OpenSpec project root or one of its subdirectories:

```sh
specview
```

You can also specify the project to open:

```sh
specview '/path/to/my project'
```

On Windows:

```powershell
specview 'C:\projects\my project'
```

The server listens on `127.0.0.1`, selects an available port, prints its URL, and opens the browser. The process stays active in your terminal; use `Ctrl+C` to stop it.

| Option | Description |
| --- | --- |
| `--no-open` | Start without automatically opening the browser. |
| `--port 4321` | Choose a port; `0` selects an available port. |
| `--version` | Print the binary version. |
| `--help` | Show help. |

Place options before the path:

```sh
specview --port 4321 --no-open '/path/to/my project'
```

### Using the interface

- **Specifications**: browse project capabilities, including nested paths.
- **Changes**: read proposals, designs, tasks, deltas, and additional documents.
- **Archives**: find archived changes and their artifacts.
- **Reader**: switch between Markdown and source, collapse sections, and follow links between documents.
- **Next command**: copy the suggested command without executing it from the application.

The green **Connected** indicator (currently labeled **Connecté** in the interface) shows that file watching is active. An edit in your editor automatically refreshes the catalog and the open document. Light and dark themes are available in the sidebar.

Progress is based on checkboxes in `tasks.md`, excluding code blocks. It does not constitute validation of a change. Only one local project is viewed at a time; external stores and multi-project workspaces are not supported. Custom schemas remain browsable, but their workflow steps are not interpreted.

## Contributing

Contributions are welcome, including bug reports, interface improvements, and documentation fixes. When reporting a bug, include your `specview` version, operating system, and reproduction steps. For a substantial change, open an issue to discuss its scope before submitting a pull request.

### Setting up local development

Requirements: Git, Go **1.27.2+**, Node.js **22.15+**, npm, and `make` for the commands below. Go race detection tests also require a C compiler. Internet access is needed for the initial dependency download.

Clone this repository using GitHub’s **Code** button, then open a terminal at its root:

```sh
make build
./bin/specview --no-open .
```

`make build` installs locked dependencies with `npm ci`, checks TypeScript, compiles the frontend, and builds `bin/specview` with `CGO_ENABLED=0`.

Without `make`, for example in PowerShell, use the equivalent commands:

```powershell
npm --prefix ui ci --no-audit --no-fund
npm --prefix ui run typecheck
npm --prefix ui run build
$env:CGO_ENABLED = '0'
go build -trimpath -o bin/specview.exe ./cmd/specview
.\bin\specview.exe --no-open .
```

### Making changes

| Directory | Purpose |
| --- | --- |
| `cmd/specview/` | CLI entry point and options. |
| `internal/project/` | OpenSpec project resolution. |
| `internal/catalog/` | File reading, progress tracking, and suggested commands. |
| `internal/server/` | HTTP server and SSE notifications. |
| `internal/web/` | Embedded compiled frontend. |
| `ui/src/` | React interface, TypeScript, and styles. |

After making changes, run `make build` again and restart the binary. File watching covers the artifacts of the project being viewed; it does not rebuild the application.

To work only on Go code after compiling the frontend assets:

```sh
go run ./cmd/specview --no-open .
```

After changing the frontend, run `npm --prefix ui run build` before restarting Go. Files in `internal/web/dist/` are generated and are not tracked by Git.

### Checking your changes

```sh
make test
```

This runs Go tests with race detection, TypeScript checks, and frontend unit tests. In PowerShell, remove the build environment variable with `Remove-Item Env:CGO_ENABLED` before running race detection tests. Without `make`, run `go test -race ./...`, `npm --prefix ui run typecheck`, and `npm --prefix ui run test:unit`.

To test the actual binary in a browser, install Chromium and run the integration tests:

```sh
cd ui
npx playwright install chromium
cd ..
make integration
```

Integration tests require permission to open a local port and launch Chromium. Results and screenshots are written to `test-results/`. CI is configured to check the project on macOS, Linux, and Windows.

Before submitting a pull request, describe the problem you solved and the checks you ran. Preserve read-only access to projects and avoid introducing remote runtime dependencies.

### Building distribution binaries

```sh
make release
```

The six binaries listed in the installation table are generated in `bin/`. Users only need the executable for their platform.

### Publishing a release

The **Release specview** GitHub Actions workflow runs when a version tag such as `0.0.1` or `v0.0.1` is pushed. Commit and push the workflow and the changes you want to release before creating the tag:

```sh
git tag -a 0.0.1 -m "Release 0.0.1"
git push origin 0.0.1
```

The workflow checks out the tagged commit, builds the frontend, runs unit tests, compiles all six binaries, and runs browser integration tests on Linux. The binary version is set from the tag (`v0.0.1` reports `specview 0.0.1`). Once the checks pass, it creates a GitHub Release containing the six executables and a `SHA256SUMS` file.

For an existing tag, push this workflow to the default branch, then select **Actions → Release specview → Run workflow** and enter the tag. This builds the code at that tag, not the latest branch contents. The tag must contain the build sources and `scripts/release.go`. Releases are only published if the workflow succeeds; an existing Release is not overwritten.

Further reading: [architecture and API](docs/architecture.md), [verification results and limitations](docs/verification.md), and [OpenSpec documentation](https://openspec.dev/docs).

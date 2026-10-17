# specview

**A local web interface for reading your [OpenSpec](https://openspec.dev) projects.**

`specview` starts from your terminal and opens your project in the browser. Browse specifications, active changes, and archives, read artifacts, and track task progress. The interface refreshes when files change and suggests the next OpenSpec command to copy into your agent or terminal.

The application is read-only: your files remain the single source of truth. The frontend is embedded in a standalone Go binary that works offline.

## Installation

On macOS, install with [Homebrew](https://brew.sh):

```sh
brew install --cask herveGuigoz/tap/specview
specview --version
```

Go, Node.js and the OpenSpec CLI are not required to run specview. You only need a browser and an OpenSpec project.

### macOS security prompt

The executable is not signed with an Apple Developer ID or notarized by Apple. macOS may display a warning that Apple could not verify whether specview contains malware.

If you trust the download, open **System Settings → Privacy & Security**, click **Open Anyway** for specview, and confirm. Then run `specview .` again. See [Apple’s instructions](https://support.apple.com/guide/mac-help/mh40616/mac).

### Updating

Stop specview with `Ctrl+C`, then run:

```sh
brew update
brew upgrade --cask herveGuigoz/tap/specview
specview --version
```

### Uninstalling

Stop specview, then run:

```sh
brew uninstall --cask herveGuigoz/tap/specview
```

Your OpenSpec projects are preserved.

## Usage

From an OpenSpec project root:

```sh
specview .
```

You can also specify the project to open:

```sh
specview '/path/to/my project'
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

This runs Go tests with race detection, TypeScript checks, and frontend unit tests. Without `make`, run `go test -race ./...`, `npm --prefix ui run typecheck`, and `npm --prefix ui run test:unit`.

To test the actual binary in a browser, install Chromium and run the integration tests:

```sh
cd ui
npx playwright install chromium
cd ..
make integration
```

Integration tests require permission to open a local port and launch Chromium. Results and screenshots are written to `test-results/`. CI is configured to check the project on macOS and Linux.

Before submitting a pull request, describe the problem you solved and the checks you ran. Preserve read-only access to projects and avoid introducing remote runtime dependencies.

### Building distribution archives

With GoReleaser OSS 2.18.0 installed, run:

```sh
make release
```

This creates local snapshot archives and `SHA256SUMS` in `dist/` without publishing them.

Further reading: [OpenSpec documentation](https://openspec.dev/docs).

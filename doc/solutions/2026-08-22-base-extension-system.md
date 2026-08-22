# Walkthrough: Base Extension System

We have implemented the base extension system for StudyLoop, providing a minimal, clean foundation for discovering and running extensions from an independent `extensions/` directory.

## Changes Made

### 1. Root Extensions Directory
- **[extensions/README.md](file:///c:/Users/vishn/PROJECT/ai-tutor/extensions/README.md)**: Documentation explaining extension layout, manifest schema, and development guidance.
- **[extensions/example/manifest.json](file:///c:/Users/vishn/PROJECT/ai-tutor/extensions/example/manifest.json)**: Minimal example manifest definition:
  ```json
  {
    "id": "example",
    "name": "Example Extension",
    "version": "0.1.0",
    "runtime": "python",
    "entrypoint": "example.py"
  }
  ```
- **[extensions/example/README.md](file:///c:/Users/vishn/PROJECT/ai-tutor/extensions/example/README.md)**: Example extension guide.

### 2. Extension Package (`internal/extension/`)
- **[internal/extension/extension.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/extension.go)**:
  - `Manifest` struct with JSON bindings and `Validate()` checking required fields and path safety.
  - `Extension` model holding metadata and absolute path directory resolution.
- **[internal/extension/manager.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/manager.go)**:
  - `Manager` providing concurrent-safe discovery, listing, and ID lookup (`Get`).
  - `ResolveExtensionsDir()` supporting custom overrides, `STUDYLOOP_EXTENSIONS_DIR` environment variable, `./extensions` in dev, and `<exe_dir>/extensions` when packaged as a Windows `.exe`.
  - Graceful handling of missing extensions directory (returns empty slice, does not fail).
- **[internal/extension/runner.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/runner.go)**, **[runner_windows.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/runner_windows.go)**, & **[runner_nowindows.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/runner_nowindows.go)**:
  - Process runner utilizing standard library `os/exec.CommandContext` with cancellation support.
  - Console window suppression on Windows.

### 3. Tests
- **[internal/extension/manager_test.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/manager_test.go)**:
  - `TestValidManifestDiscovery`: Tests scanning valid extensions and extracting metadata.
  - `TestMissingExtensionsDir`: Verifies missing directory is handled gracefully.
  - `TestInvalidManifestHandling`: Verifies malformed JSON, missing fields, traversal IDs, and non-directories are handled cleanly.
  - `TestManifestValidation`: Unit tests for manifest field validator.
- **[internal/extension/runner_test.go](file:///c:/Users/vishn/PROJECT/ai-tutor/internal/extension/runner_test.go)**:
  - `TestRunnerExecution`: Verifies executing a command and capturing output.
  - `TestRunnerContextCancellation`: Verifies context cancellation terminates external processes.
  - `TestRunnerNilExtension`: Verifies nil check.

---

## Verification Results

### Automated Tests
```powershell
go test -v ./internal/extension/...
```
Output:
```text
=== RUN   TestValidManifestDiscovery
--- PASS: TestValidManifestDiscovery (0.18s)
=== RUN   TestMissingExtensionsDir
--- PASS: TestMissingExtensionsDir (0.01s)
=== RUN   TestInvalidManifestHandling
--- PASS: TestInvalidManifestHandling (0.29s)
=== RUN   TestManifestValidation
--- PASS: TestManifestValidation (0.00s)
=== RUN   TestRunnerExecution
--- PASS: TestRunnerExecution (0.20s)
=== RUN   TestRunnerContextCancellation
--- PASS: TestRunnerContextCancellation (0.07s)
=== RUN   TestRunnerNilExtension
--- PASS: TestRunnerNilExtension (0.00s)
PASS
ok  	ai-tutor/internal/extension	3.256s
```

### Full Repository Test Suite
```powershell
go test ./...
```
Output:
```text
ok  	ai-tutor/internal/app	(cached)
ok  	ai-tutor/internal/db	(cached)
ok  	ai-tutor/internal/embeddings	(cached)
ok  	ai-tutor/internal/extension	1.971s
ok  	ai-tutor/internal/llm	(cached)
ok  	ai-tutor/internal/notebook	(cached)
ok  	ai-tutor/internal/runtime	(cached)
ok  	ai-tutor/internal/scheduler	(cached)
ok  	ai-tutor/internal/study	(cached)
ok  	ai-tutor/internal/utils	(cached)
```

### Build Verification
```powershell
go build ./...
```
Build succeeded with code 0.

# CLAUDE.md

This document provides Claude-specific best practices and instructions for working on the `autoupdate` project.

## Interaction Style
- **Chain of Thought (CoT):** Always perform a thorough analysis before implementing changes. State your assumptions and identified risks clearly.
- **Surgical Edits:** Claude is excellent at targeted refactoring. Prefer making precise edits to specific lines rather than rewriting entire files.
- **Verification First:** Before fixing a bug, attempt to reproduce it with a test case.

## Go Best Practices for Claude
- **Strong Typing:** Leverage Go's type system. Avoid `interface{}` where possible.
- **Error Handling:** Claude should always check returned errors. Use `errors.Is` and `errors.As` for modern error inspection.
- **Context:** Use `context.Context` for cancellation and timeouts in network or long-running operations (like the Watcher).
- **Concurrency:** Be extremely careful with race conditions. Always use `-race` when running tests for Claude-generated code.

## SQLite & Storage Safety
- **WAL Mode:** Ensure all database interactions respect the Write-Ahead Logging configuration.
- **Connection Management:** Do not open/close databases in a loop. Use the connection cache implemented in `server/storage/blob.go`.
- **Serialization:** SQLite writes must be serialized using `SetMaxOpenConns(1)`.

## Project Specifics
- **Library First:** Always prioritize the client-side experience. The library must remain lightweight and easy to embed.
- **Source Agnosticism:** When adding features to the Watcher, ensure they work across all sources (GitHub, GitLab, Gitea, etc.).
- **Watcher Lifecycle:** The `UpdateWatcher` has a complex lifecycle (started -> update -> restart -> stopped). Ensure transitions are handled gracefully without leaking goroutines.
- **Self-Update Scripts:** Modifying `exec_unix.go` or `exec_windows.go` requires platform-specific knowledge. Claude should double-check syscalls and process signals.
- **Optional Server:** The `autoupdate/server` is a secondary component. Improvements here should not break client compatibility.

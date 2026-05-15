# AGENTS.md

This document provides AI coding agents (Gemini, Claude, Codex) with the context and commands needed to work effectively on the `autoupdate` codebase.

## Project Overview
- **Core Identity:** A Golang client library for binary auto-updates.
- **Primary Use Case:** Embedding `autoupdate.Watch()` into CLI tools or services to enable self-updating from GitHub, GitLab, Gitea, or custom servers.
- **Key Components:**
    - **UpdateWatcher (Core):** The library's background worker that monitors sources and orchestrates updates.
    - **Update Sources:** Implementations for GitHub, GitLab, Gitea, and Generic (REST) providers.
    - **Self-Update Scripts:** Platform-native tools for binary replacement and process restart.
    - **Optional Server:** A reference implementation of a self-hosted update server (`autoupdate/server`).

## Setup & Build Commands
- **Check Environment:** `go version` (Expects 1.26.1 via gvm)
- **Download Dependencies:** `go mod download`
- **Build Server:** `go build ./cmd/autoupdate-server/...`
- **Build Examples:** `go build ./cmd/examples/...`
- **Run Lint (if available):** `go vet ./...`

## Code Style & Conventions
- **Formatting:** Standard `go fmt` is required.
- **Concurrency:** All runtime behavior MUST be goroutine-safe. Use `sync.RWMutex` for state protection.
- **Database:** SQLite MUST use WAL mode and connection serialization (`SetMaxOpenConns(1)`).
- **Errors:** Use idiomatic Go error handling. Wrap errors with context where appropriate: `fmt.Errorf("failed to X: %w", err)`.
- **Naming:** Follow Go standard naming (CamelCase).
- **Surgical Edits:** Prefer minimal, targeted changes to existing logic.

## Testing Instructions
- **Run All Tests:** `go test ./...`
- **Run Specific Test:** `go test -v -run TestName .`
- **Race Detection:** `go test -race ./...` (Crucial for the Watcher)

## Project Context & Architecture
- **Watcher Entrypoint:** `autoupdate.Watch(config)` in `autoupdate.go`.
- **Server Entrypoint:** `server.New(config)` in `server/server.go`.
- **Self-Update Logic:** Platform-specific scripts in `exec_unix.go` and `exec_windows.go`.
- **Storage Logic:** Per-app SQLite databases in `server/storage/blob.go`.

## PR & Commit Guidelines
- **Format:** `feat: <description>` or `fix: <description>`.
- **Requirement:** Ensure all tests pass and `go build` succeeds before finalizing.
- **Verification:** Always verify SQLite write safety if modifying storage logic.

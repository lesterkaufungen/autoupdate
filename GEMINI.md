# GEMINI.md

For cross-agent instructions, see [AGENTS.md](./AGENTS.md). For Claude-specific guidelines, see [CLAUDE.md](./CLAUDE.md).

## Project Context
- **Core Focus:** Minimal, production-safe, event-driven binary updates.
- **Primary Agent:** Gemini CLI (Interactive CLI agent).
- **Key Concepts:** Golang, Self-Update, Binary Distribution, SQLite WAL, Thread-Safe Watcher, Process Management, Gin/Echo Update Server.

# Current Machine
...

Current machine doesn't have default locatable go binary, current machine have gvm, before I start gemini, I always select needed golang version using gvm use go1.26.1, so it should be transparent for you.

# Project Overview

This project is primarily a **Golang client library** for automatic binary updates and process lifecycle management. It allows applications to monitor remote sources, download updates, and restart themselves or managed child processes.

The library is source-agnostic and supports:
- **Public/Private Providers:** GitHub, GitLab, and Gitea.
- **Self-Hosted:** Any custom server following the [REST API Specification](#rest-api-specification). An optional, robust update server is included in this module for self-hosting needs.

The core functionality revolves around two primary workflows:
1. **Self-Update:** Watch and update the current application binary.
2. **External Watcher:** Run and manage external binaries as a supervisor.

The package is designed to provide a minimal, thread-safe, event-driven API for embedding automatic update behavior into CLI tools and long-running services.

The core API revolves around a single entrypoint:

```go
autoupdate.Watch(config)
```

---

# Design Goals

* Minimal integration effort.
* Safe process restart behavior.
* Thread-safe runtime operations.
* Cross-platform support where possible.
* Event-driven update handling.
* Clean API for CLI tools and services.
* Graceful process lifecycle management.
* Non-blocking subscriptions.
* Streaming communication with child process stdin.
* Predictable and composable abstractions.

Avoid overengineering.

The API should remain understandable within minutes.

---

# Core API

## Watch

```go
watcher, err := autoupdate.Watch(config)
```

`Watch` initializes the runtime watcher, starts update monitoring, and optionally launches or manages a target process.

---

# WatchConfig

```go
type WatchConfig struct {
    // Source defines where to fetch updates from.
    Source Source

    // Version is the current version of the application.
    Version string

    // ServiceName is the name of the system service (systemd, launchd, or Windows Service).
    // If set, the update script will restart the service instead of the binary.
    ServiceName string

    // UpdateServer is the base endpoint of the update server.
    // Deprecated: Use Source instead.
    UpdateServer string

    // Headers contains HTTP headers included with update requests.
    Headers map[string]string

    // Interval defines how frequently updates are checked.
    Interval time.Duration

    // Timeout defines the maximum duration for update requests.
    Timeout time.Duration

    // Target is the external executable path to launch and monitor.
    // Leave empty when using SelfUpdate.
    Target string

    // Args are arguments passed to the target executable.
    Args []string

    // SelfUpdate enables self-watching/self-updating mode.
    SelfUpdate bool

    // AutoRestart automatically restarts after a successful update.
    AutoRestart bool

    // Callbacks registers lifecycle/update hooks during initialization.
    Callbacks Callbacks
}
```

---

# Update Sources

The module supports multiple update providers through the `Source` interface.

## GitHub Source
Fetches updates from GitHub Releases. Automatically matches assets based on the current OS and architecture.

```go
config := autoupdate.WatchConfig{
    Source:  autoupdate.NewGitHubSource("my-org", "my-app", os.Getenv("GITHUB_TOKEN")),
    Version: "1.0.0",
}
```

## GitLab Source
Fetches updates from GitLab Releases using the GitLab API.

```go
config := autoupdate.WatchConfig{
    Source:  autoupdate.NewGitLabSource("https://gitlab.com", "12345678", os.Getenv("GITLAB_TOKEN")),
    Version: "1.0.0",
}
```

## Gitea Source
Fetches updates from a Gitea instance.

```go
config := autoupdate.WatchConfig{
    Source:  autoupdate.NewGiteaSource("https://gitea.example.com", "my-org", "my-app", os.Getenv("GITEA_TOKEN")),
    Version: "1.0.0",
}
```

## Generic Source (Default)
Connects to a dedicated `autoupdate` server. Recommended for custom binary distribution.

```go
config := autoupdate.WatchConfig{
    Source:  autoupdate.NewGenericSource("https://updates.example.com", "app-123", map[string]string{
        "X-App-Token": "my-secret-token",
    }),
    Version: "1.0.0",
}
```

---

# Execution Modes

## External Binary Mode
Launch and manage another executable:

```go
watcher, err := autoupdate.Watch(autoupdate.WatchConfig{
    Target:        "./my-service",
    Args:          []string{"--port=8080"},
    Source:        autoupdate.NewGenericSource("https://updates.example.com", "app-123", nil),
    Interval:      time.Minute,
    AutoRestart:   true,
})
```

## Self-Watching Mode
Watch and update the current application binary:

```go
watcher, err := autoupdate.Watch(autoupdate.WatchConfig{
    SelfUpdate:   true,
    Source:       autoupdate.NewGenericSource("https://updates.example.com", "app-123", nil),
    Interval:     time.Minute,
    AutoRestart:  true,
})
```

---

# UpdateWatcher

`UpdateWatcher` is the central runtime controller returned by `Watch()`.

```go
type UpdateWatcher interface {
    Watch() <-chan UpdateEvent
    OnUpdate(func(UpdateEvent))
    Pipe() LauncherPipe
    Restart() error
    Shutdown() error
}
```

---

# Subscription API

## Channel Subscription
```go
ch := watcher.Watch()
for event := range ch {
    fmt.Println(event.Type)
}
```

## Callback Subscription
```go
watcher.OnUpdate(func(event UpdateEvent) {
    switch event.Type {
    case autoupdate.EventUpdate:
        fmt.Printf("New version available: %s\n", event.Version)
    case autoupdate.EventRestart:
        fmt.Println("Application is restarting...")
    case autoupdate.EventError:
        fmt.Printf("Update error: %v\n", event.Error)
    }
})
```

---

# Events

```go
type UpdateEvent struct {
    Type      UpdateEventType
    Version   string
    Timestamp time.Time
    Error     error
}

const (
    EventStarted UpdateEventType = "started"
    EventUpdate  UpdateEventType = "update"
    EventRestart UpdateEventType = "restart"
    EventStopped UpdateEventType = "stopped"
    EventError   UpdateEventType = "error"
)
```

---

# LauncherPipe
Provides streaming communication into the managed process stdin/stdout.

```go
pipe := watcher.Pipe()
pipe.Write([]byte("reload\n"))
```

---

# Update Mechanism (Self-Update)

1. **Download:** Binary is downloaded and verified (SHA-256).
2. **Spawn Script:** Embedded platform-specific script (`update.sh` or `update.ps1`) is written and executed detached.
3. **Detach & Exit:** Main application exits.
4. **Kill & Replace:** Script terminates the old process, replaces binary, and restarts.
5. **Restart Strategy:** Restarts binary directly or via system service (`ServiceName`).
6. **Cleanup:** Script deletes itself.

---

# Update Server Module (`autoupdate/server`)

Robust standalone and embeddable update server.

## Features
* **Multi-Tenancy:** Multiple users and applications.
* **Authentication:** Per-app tokens and `X-Admin-Token` for management.
* **Analytics:** Tracks node-linked unique users, retention, and GeoIP distribution.
* **Storage:** SQLite Blob strategy with WAL mode for write-safety and performance.

## Framework Integration

```go
// Standard HTTP
srv, err := autoupdate.HTTP(http.DefaultServeMux, server.Config{
    DatabasePath: "./autoupdate.db",
    StoragePath:  "./binaries",
    Admin: server.NewAdmin("admin", "pass"),
    App:   server.NewApp("My App"),
})

// Gin
autoupdate.Gin(router, config)

// Echo
autoupdate.Echo(group, config)
```

---
# REST API Specification

## Authentication

### Admin
* **Header:** `X-Admin-Token: <token>`

## Client Endpoints

### Check for Update
`GET /update`

**Required Headers:**
* `X-App-ID`: Application UUID
* `X-OS`: Client OS
* `X-Arch`: Client Arch
* `X-Version`: Current Version
* `X-Node-ID`: Hardware ID

**Response:**
```json
{
  "update_available": true,
  "version": "1.2.0",
  "url": "https://example.com/download/app-123/456",
  "sha256": "hash",
  "required": false
}
```

### Download Binary
`GET /download/:app_id/:release_id`

---

## Management API (`/admin/*`)

### Applications
* `GET /admin/apps`: List applications.
* `POST /admin/apps`: Create application (`{"name": "...", "description": "..."}`).

### Releases
* `GET /admin/releases`: List releases.
* `POST /admin/releases`: Upload binary (`multipart/form-data`).

### Logs & Stats
* `GET /admin/logs?app_id=:id&from=:t1&to=:t2`: Raw interaction logs.
* `GET /admin/stats?app_id=:id&from=:t1&to=:t2`: Aggregated retention stats.

#### Stats Retention Logic
* **New:** First recorded interaction falls within the range.
* **Active:** Interaction falls within the range, but first interaction was earlier.

---

# Concurrency & Safety

* **Thread-Safe:** All watcher and storage operations are goroutine-safe.
* **Write-Safety:** SQLite uses WAL mode and connection serialization for robust concurrent writes.
* **Validation:** All binaries are verified via SHA-256 before execution.

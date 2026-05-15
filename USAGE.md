# autoupdate Usage Guide

This guide provides comprehensive examples and detailed instructions on how to integrate the `autoupdate` library into your Go applications, both as a client and as a server.

---

## Client-Side Integration

The client-side `Watcher` is a background process that monitors a remote source for new versions of an executable. When an update is detected, it can download, verify, and automatically restart the application.

### 1. Update Sources

The library supports multiple remote sources through the `Source` interface.

#### GitHub Source
Fetches updates from GitHub Releases. It automatically matches assets based on the current OS and architecture.

```go
source := autoupdate.NewGitHubSource(
    "google",           // Owner
    "go-github",        // Repository
    os.Getenv("GH_TOKEN"), // Optional: Token for private repos or rate limiting
)

watcher, _ := autoupdate.Watch(autoupdate.WatchConfig{
    Source:     source,
    Version:    "1.2.0",
    SelfUpdate: true,
})
```

#### GitLab Source
Fetches updates from GitLab Releases using the GitLab API.

```go
source := autoupdate.NewGitLabSource(
    "https://gitlab.com", // Base URL
    "12345678",           // Project ID
    os.Getenv("GL_TOKEN"), // Personal Access Token
)
```

#### Gitea Source
Fetches updates from a Gitea instance.

```go
source := autoupdate.NewGiteaSource(
    "https://gitea.com",
    "my-user",
    "my-repo",
    os.Getenv("GITEA_TOKEN"),
)
```

#### Generic Source (Default)
Connects to a dedicated `autoupdate` server. This is the recommended choice for custom binary distribution.

```go
source := autoupdate.NewGenericSource(
    "https://updates.example.com",
    "app-unique-uuid",
    nil, // No headers needed for public access
)
```

### 2. Execution Modes

#### Self-Update Mode
In this mode, the library watches the currently running binary. When an update is applied, it spawns a detached script that replaces the binary and restarts the process.

```go
config := autoupdate.WatchConfig{
    SelfUpdate:   true,
    AutoRestart:  true,
    Version:      "1.0.0",
    Interval:     30 * time.Minute,
    Source:       source,
}
```

#### External Binary Mode
Use this to manage a child process. The watcher will launch the binary and ensure it stays updated.

```go
config := autoupdate.WatchConfig{
    Target:      "./bin/worker",
    Args:        []string{"--verbose"},
    AutoRestart: true,
    Source:      source,
}
```

---

## Server-Side Integration

The `autoupdate/server` provides a robust backend for storing binaries and managing releases. It can be integrated into your existing web server with a single line.

### 1. Bootstrapping (Initial Setup)

To avoid manual database entry, use the bootstrap system to create an initial admin and application on startup.

```go
import "github.com/lesterkaufungen/autoupdate/server"

// 1. Create an admin user
admin := server.NewAdmin("admin", "secure-password-123")

// 2. Create a default app (ID and Token are generated automatically)
app := server.NewApp("Main API")

config := server.Config{
    Admin: admin,
    App:   app,
}
```

### 2. Framework-Specific Integration

#### Gin Integration
Registers all update and management endpoints on a Gin router.

```go
import "github.com/lesterkaufungen/autoupdate"

r := gin.Default()

// Registers /update/*, /download/* and /admin/*
autoupdate.Gin(r, config)

r.Run(":8080")
```

#### Echo Integration
Registers the server on an Echo router or group.

```go
import "github.com/lesterkaufungen/autoupdate"

e := echo.New()

// Registers endpoints on the root group
autoupdate.Echo(e.Group(""), config)

e.Start(":8080")
```

#### Standard net/http Integration
Registers the server on a standard `http.ServeMux`.

```go
import "github.com/lesterkaufungen/autoupdate"

mux := http.NewServeMux()

autoupdate.HTTP(mux, config)

http.ListenAndServe(":8080", mux)
```

---

## Management & Monitoring

Once your server is running, you can manage applications and monitor updates through the `/admin/` endpoints. All admin requests require the `X-Admin-Token` header.

### 1. Tracking Activity (Logs)
The `/admin/logs` endpoint provides a real-time stream of all client update checks and downloads. You can use this to troubleshoot connectivity or monitor download rates.

**Example: Fetch logs for May 2026**
`GET /admin/logs?app_id=my-app&from=2026-05-01T00:00:00Z&to=2026-05-31T23:59:59Z`

### 2. Retention and Growth (Stats)
The `/admin/stats` endpoint provides high-level insights into your user base. It automatically tracks unique nodes using an internal hardware-linked ID.

*   **New Users**: Nodes seen for the very first time in the given period.
*   **Active Users**: Returning nodes that were already known before the period started.
*   **Geographic Breakdown**: Automatically resolves client IPs to countries (requires GeoIP database).

**Example: Get weekly stats breakdown**
`GET /admin/stats?app_id=my-app&from=2026-05-01T00:00:00Z&to=2026-05-07T23:59:59Z`

### 3. Managing Applications & Releases

You can manage your software products directly from the Applications dashboard:

*   **View Metadata:** Application cards show the unique ID, creation date, latest version, and total release count. You can also set a custom icon URL for each application.
*   **Edit Application:** Update the name, description, and icon of an existing application.
*   **Quick Release:** Upload new binaries for multiple platforms (Linux, macOS, Windows) and architectures (amd64, arm64) in a single request directly from the application card.

---

## Event Handling
...
The `UpdateWatcher` emits events throughout its lifecycle. You can subscribe to these events to perform custom logic (e.g., logging or graceful cleanup).

```go
watcher, _ := autoupdate.Watch(config)

go func() {
    for event := range watcher.Watch() {
        switch event.Type {
        case autoupdate.EventUpdate:
            log.Printf("New version available: %s", event.Version)
        case autoupdate.EventRestart:
            log.Println("Application is restarting...")
        case autoupdate.EventError:
            log.Printf("Update error: %v", event.Error)
        }
    }
}()
```

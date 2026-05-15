# autoupdate REST API Reference

This document describes the HTTP API provided by the `autoupdate` server. This API can be used to develop custom client watchers or headless management UIs.

## Authentication

### Admin Authentication
Requests to the management API (`/admin/*`) must provide the master admin token.
*   **Header:** `X-Admin-Token: <token>`

---

## Client Endpoints

### Check for Update
`GET /update`

Returns whether an update is available for the specified platform and version. This is a public endpoint.

**Required Headers:**
* `X-App-ID`: Application UUID.
* `X-OS`: Client operating system (linux, darwin, windows).
* `X-Arch`: Client architecture (amd64, arm64).
* `X-Version`: Current client version.
* `X-Node-ID`: Unique machine identifier (for analytics).

**Response (Update Available):**
```json
{
  "update_available": true,
  "version": "1.2.0",
  "url": "https://example.com/download/app-123/456",
  "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "required": false
}
```

### Download Binary
`GET /download/:app_id/:release_id`

Streams the binary file for the specified release. This is a public endpoint.

---

## Management API (`/admin/*`)

All endpoints in this section require `X-Admin-Token`.

### Applications

#### List Applications
`GET /admin/apps`

**Response:**
```json
[
  {
    "id": "uuid-string",
    "name": "App Name",
    "description": "...",
    "icon": "https://example.com/icon.png",
    "min_version": "1.0.0",
    "created_at": "...",
    "latest_version": "1.2.0",
    "last_release_at": "timestamp",
    "release_count": 5,
    "today_installed": 10,
    "total_installed": 150
  }
]
```

#### Create Application
`POST /admin/apps`

**Body:**
```json
{
  "name": "My New App",
  "description": "Optional description",
  "icon": "Optional icon URL"
}
```

#### Update Application
`PUT /admin/apps`

**Body:**
```json
{
  "id": "uuid-string",
  "name": "Updated Name",
  "description": "Updated description",
  "icon": "Updated icon URL"
}
```

---

### Releases

#### List Releases
`GET /admin/releases`

#### Upload Release
`POST /admin/releases`

Supports uploading multiple binaries in a single request using `multipart/form-data`.

**Form Fields:**
*   `app_id`: (string) Target application ID.
*   `version`: (string) Version string (e.g., "1.2.0").
*   `scheduled_at`: (string, optional) RFC3339 timestamp for scheduled activation.
*   `binary_linux_amd64`: (file) Binary file for Linux amd64.
*   `binary_windows_amd64`: (file) Binary file for Windows amd64.
*   `binary_darwin_arm64`: (file) Binary file for macOS arm64.
*   `binary`: (file) Legacy field for single binary upload (requires `os` and `arch` fields).

**Response:**
```json
[
  {
    "id": 123,
    "version": "1.0.1",
    "os": "linux",
    "arch": "amd64",
    "sha256": "...",
    "size": 12345
  }
]
```

---

### Dashboard

#### Get Dashboard Summary
`GET /admin/dashboard/stats`

Returns high-level global statistics across all applications.

**Response:**
```json
{
  "today_installed": 15,
  "total_installed": 1250,
  "total_downloads": 840,
  "total_nodes": 1250
}
```

#### Get Traffic Trends
`GET /admin/dashboard/trends?app_id=:app_id&days=:days`

Returns daily aggregated traffic data (checks and downloads) for the last N days.

**Query Parameters:**
*   `app_id` (optional): Filter trends by specific application ID.
*   `days` (optional, default 30): Number of days to return.

**Response:**
```json
[
  {
    "date": "2026-05-01",
    "checks": 150,
    "downloads": 12
  },
  {
    "date": "2026-05-02",
    "checks": 165,
    "downloads": 8
  }
]
```

---

### Logs & Stats

#### List Logs
`GET /admin/logs?app_id=:app_id&from=:from&to=:to`

Returns a raw log of all client interactions (update checks and downloads).

**Query Parameters:**
*   `app_id` (optional): Filter logs by specific application ID.
*   `from` (optional): Filter logs starting from this RFC3339 timestamp (inclusive).
*   `to` (optional): Filter logs up to this RFC3339 timestamp (inclusive).

**Example Request:**
`GET /admin/logs?app_id=app-123&from=2026-05-01T00:00:00Z&to=2026-05-10T23:59:59Z`

**Response Fields:**
*   `node_id`: Unique identifier for the client machine.
*   `action`: Either `check` (update check) or `download`.
*   `country`: Two-letter ISO country code resolved from IP.
*   `ip`: Obfuscated or raw client IP address.

---

#### Get Application Stats
`GET /admin/stats?app_id=:app_id&from=:from&to=:to`

Returns high-level aggregated statistics grouped by application version. This endpoint is designed for building dashboards and tracking user retention.

**Query Parameters:**
*   `app_id` (**required**): Target application ID.
*   `from` (optional): Start of the reporting period (RFC3339).
*   `to` (optional): End of the reporting period (RFC3339).

**Retention Logic:**
The server uses the `node_id` and the requested date range to distinguish between user types:
*   **New**: A node is considered "New" if its very first recorded interaction with the server falls within the specified `from`/`to` range. If no range is provided, "New" refers to nodes seen for the first time today (UTC).
*   **Active**: A node is considered "Active" if it has at least one interaction within the requested range, but its first-ever interaction occurred *before* the start of that range.

**Response Structure:**
The response is an array of objects, one for each version seen during the period.

```json
[
  {
    "version": "1.2.0",
    "new": 2,
    "active": 8,
    "platforms": {
      "linux-amd64": { "new": 1, "active": 4 },
      "darwin-arm64": { "new": 1, "active": 4 }
    },
    "countries": {
      "US": {
        "linux-amd64": { "new": 1, "active": 2 },
        "darwin-arm64": { "new": 0, "active": 2 }
      }
    }
  }
]
```

**Key Definitions:**
*   **`active` (Top Level)**: Total returning unique nodes on this version.
*   **`new` (Top Level)**: Total first-time unique nodes on this version.
*   **`platforms`**: Distribution of New/Active nodes across OS/Arch combinations.
*   **`countries`**: Geographic distribution. Each country code contains its own platform breakdown of New/Active nodes.

**Example Curl:**
```bash
curl -H "X-Admin-Token: your-token" \
     "https://updates.example.com/admin/stats?app_id=my-app&from=2026-05-01T00:00:00Z"
```




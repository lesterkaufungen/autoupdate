# Management Dashboard Reference

The `autoupdate` server includes a built-in, real-time management dashboard for overseeing application distribution and monitoring client activity.

## Accessing the Dashboard
- **URL:** `http://<your-server>/ui/`
- **Authentication:** Requires a valid username and password (configured via `server.Config` or environment variables).

---

## 🔐 Authentication
The dashboard features a secure login screen. Upon successful authentication, a secure session token (Admin Token) is stored locally to authorize subsequent API and WebSocket requests.

- **Login:** Enter administrative credentials to enter.
- **Logout:** Use the logout button in the bottom-left sidebar to terminate the session.

---

## 📊 Dashboard Overview
The landing page provides a comprehensive real-time overview of the update ecosystem:
- **Key Metrics:** High-level cards showing total applications, active unique nodes, total downloads, and recent updates with trend indicators.
- **Traffic Trends:** A large interactive line chart visualizing update checks and successful downloads over the last 14 days.
- **Active Applications:** An information-dense table showing current status, latest versions, and reach (unique nodes) for each application.
- **Recent Activity:** A feed of the most recent binary releases across all applications.

---

## 📦 Application Management
Located at `/ui/apps`, this section allows you to manage the software products you distribute.
- **Create Application:** Modal interface to define a new app with a name, description, and custom icon.
- **Application Cards:** Detailed cards showing release counts, latest version info, and install statistics (Today vs. Total).
- **Management:** Edit application metadata or trigger a new release upload directly from the card.

---

## 🚀 Release Management
Located at `/ui/releases`, this is where you manage binary distribution.
- **Release Table:** A comprehensive list of all binaries, including version, platform (OS/Arch), file size, and upload timestamp.
- **Upload Release:** A multipart upload interface to add new binaries.
  - **App Selection:** Choose which application this release belongs to.
  - **Version:** Define the semantic version (e.g., `1.2.0`).
  - **Platform:** Specify target OS (Linux, macOS, Windows) and Architecture (amd64, arm64).
  - **Binary Upload:** Drag-and-drop or file selection for the executable.

---

## 📈 Statistics & Analytics
Located at `/ui/stats`, this page provides deep insights into user retention and distribution.
- **Visual Analytics:** Interactive charts for platform distribution (Doughnut) and version adoption (Stacked Bar).
- **Node Retention:**
  - **New Nodes:** Unique client machines seen for the first time in the selected period.
  - **Active Nodes:** Returning client machines that have interacted previously.
- **Geographic Distribution:** Flag-based breakdown of regional adoption and node growth.
- **Version Adoption:** Detailed version adoption metrics showing distribution across your user base.

---

## 📜 Activity Logs
Located at `/ui/logs`, this provides a real-time stream of all client-server interactions.
- **Events:** Tracks `check` (update checks) and `download` (binary fetches).
- **Node Metadata:** Displays Node IDs, client IP addresses (obfuscated), and geographic location (using GeoIP).
- **Live Stream:** Automatically prepends new events as they occur via WebSockets.

---

## 📡 Real-time Synchronization
The dashboard utilizes WebSockets to stay "alive":
- **Live Logs:** New activity appears instantly without refreshing.
- **Auto-Refreshing Stats:** Statistics update automatically when new client events are detected.
- **Resilient Connection:** Automatic reconnection logic handles server restarts or temporary network drops.

# Frontend Specification

## Overview
The `autoupdate` server includes a built-in management dashboard for managing applications, releases, and viewing analytics. The frontend is built with Svelte 5 and Tailwind CSS 4, utilizing `shadcn-svelte` for UI components.

## Tech Stack
- **Framework:** [Svelte 5](https://svelte.dev/) (Runes-based)
- **Styling:** [Tailwind CSS 4](https://tailwindcss.com/)
- **Components:** [shadcn-svelte](https://shadcn-svelte.com/)
- **Icons:** [Lucide Svelte](https://lucide.dev/guide/svelte)
- **Build Tool:** Vite
- **Language:** TypeScript

## Directory Structure
`server/frontend/`
- `src/`: Svelte source code.
- `static/`: Static assets.
- `dist/`: Compiled assets (embedded into Go).

## Integration
The frontend is embedded into the Go binary using `go:embed`. It is served by the `autoupdate` server when `server.Config.UI` is enabled. See [DASHBOARD.md](./DASHBOARD.md) for a detailed user guide.

### Routes
- `/ui/`: Dashboard Overview
- `/ui/apps`: Application Management
- `/ui/releases`: Release Management
- `/ui/stats`: Analytics and Statistics
- `/ui/logs`: Raw Interaction Logs

### API Communication
The frontend communicates with the server via the `/admin` endpoints defined in `API.md`.

## Build Process
To build the frontend for production:
```bash
cd server/frontend
npm install
npm run build
```
The resulting `dist/` directory is then used by the Go server for embedding.

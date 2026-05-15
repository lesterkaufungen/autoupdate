#!/bin/bash
set -e

# Configuration
PORT=${PORT:-8080}
ADMIN_TOKEN=${ADMIN_TOKEN:-"admin-secret-token"}
APP_NAME=${APP_NAME:-"Tesla Fleet Manager"}
APP_ICON=${APP_ICON:-"https://upload.wikimedia.org/wikipedia/commons/e/e8/Tesla_logo.png"}

# Fresh start: remove existing database
rm -f autoupdate.db autoupdate.db-shm autoupdate.db-wal

echo "🚀 Building frontend..."
cd server/frontend
npm install --silent
npm run build
cd ../..

echo "🔨 Building server..."
go build -o autoupdate-server ./cmd/server/main.go

echo "🏃 Starting server on http://localhost:$PORT/ui/"

export PORT=$PORT
export ADMIN_TOKEN=$ADMIN_TOKEN
export ENABLE_UI=true
export DEMO=true
export BOOTSTRAP_USER="admin"
export BOOTSTRAP_PASS="admin"
export BOOTSTRAP_APP_NAME="$APP_NAME"
export BOOTSTRAP_APP_ICON="$APP_ICON"

./autoupdate-server

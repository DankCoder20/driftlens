#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
python3 scripts/init-env.py
set -a
. ./.env
set +a
export DRIFTLENS_DATA_DIR="$PWD/.data"
export DRIFTLENS_REPOSITORY_ROOT="$PWD/demo"
export DRIFTLENS_CREDENTIAL_ROOT="$PWD/credentials"
export DRIFTLENS_FRONTEND_DIR="$PWD/frontend/dist"
(cd frontend && npm ci && npm run build)
echo 'Open http://127.0.0.1:8080. The API token is in .env.'
cd backend
exec go run ./cmd/server

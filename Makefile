.PHONY: dev test build demo-up demo-down compose
dev:
	sh scripts/dev.sh
test:
	cd backend && go test ./...
	cd frontend && npm ci && npm test
build:
	cd backend && go build ./cmd/server
	cd frontend && npm ci && npm run build
demo-up:
	python3 scripts/demo.py up
demo-down:
	python3 scripts/demo.py down
compose:
	python3 scripts/init-env.py
	docker compose --env-file .env -f deployments/docker-compose.yml up --build -d

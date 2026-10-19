.PHONY: generate renderer-setup test test-storage render-demo render-db seed-catalog run ngrok-login run-telegram

generate:
	go generate ./internal/storage
	go generate ./internal/web/views

run:
	go run ./cmd/app

ngrok-login:
	bash scripts/telegram-dev.sh login

run-telegram:
	bash scripts/telegram-dev.sh run

render-db:
	go run ./cmd/renderdb

seed-catalog:
	go run ./cmd/seedcatalog

test-storage:
	@test -n "$$TEST_DATABASE_URL" || (echo 'Set TEST_DATABASE_URL for an isolated PostgreSQL test schema'; exit 1)
	go test -tags integration ./internal/storage/... -count=1

renderer-setup:
	uv venv --allow-existing --python python3 .venv
	uv pip sync --python .venv/bin/python python/requirements.txt

test:
	go test ./...
	.venv/bin/python -m unittest discover -s python/renderer -p 'test_*.py'

render-demo:
	go run ./cmd/renderdemo

#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
task_ngrok="${NGROK_BIN:-$HOME/.local/bin/ngrok}"
if [[ ! -x "$task_ngrok" ]]; then
    task_ngrok="$(command -v ngrok || true)"
fi

if [[ "${1:-run}" == login ]]; then
	if [[ -z "$task_ngrok" || ! -x "$task_ngrok" ]]; then
		echo 'Установи ngrok: https://ngrok.com/download/linux' >&2
		exit 1
	fi
    read -r -s -p 'Authtoken из https://dashboard.ngrok.com/get-started/your-authtoken: ' task_token
    printf '\n'
    if [[ -z "$task_token" ]]; then
        echo 'Токен не введён.' >&2
        exit 1
    fi
    umask 077
    "$task_ngrok" config add-authtoken "$task_token"
    unset task_token
    exit 0
fi
if [[ "${1:-run}" != run ]]; then
    echo 'Использование: telegram-dev.sh login|run' >&2
    exit 1
fi

if [[ -f .env ]]; then
    set -a
    source ./.env
    set +a
fi
if [[ "${APP_URL:-}" != https://* ]]; then
    echo 'Укажи в .env APP_URL=https://твой-dev-домен из https://dashboard.ngrok.com/domains' >&2
    exit 1
fi
task_tunnel_provider="${TUNNEL_PROVIDER:-ngrok}"
if [[ "$task_tunnel_provider" != ngrok && "$task_tunnel_provider" != tailscale ]]; then
    echo 'TUNNEL_PROVIDER должен быть ngrok или tailscale.' >&2
    exit 1
fi
if [[ "$task_tunnel_provider" == ngrok && ( -z "$task_ngrok" || ! -x "$task_ngrok" ) ]]; then
	 echo 'Установи ngrok: https://ngrok.com/download/linux' >&2
	 exit 1
fi
task_session_secret="${SESSION_SECRET:-}"
if [[ -z "${TELEGRAM_BOT_TOKEN:-}" || ${#task_session_secret} -lt 32 ]]; then
    echo 'Нужны TELEGRAM_BOT_TOKEN и SESSION_SECRET минимум 32 символа в .env.' >&2
    exit 1
fi
unset task_session_secret

export DEV_LOGIN=false SERVER_HOST=127.0.0.1 SERVER_PORT="${SERVER_PORT:-8080}"
python3 - "$SERVER_PORT" <<'PY'
import socket, sys
try:
    with socket.create_connection(('127.0.0.1', int(sys.argv[1])), timeout=1):
        sys.exit('Порт занят. Останови прежний make run через Ctrl+C или выбери другой SERVER_PORT.')
except OSError:
    pass
PY

task_runtime="$(mktemp -d /tmp/contgen-telegram.XXXXXX)"
task_app_pid=''
task_tunnel_pid=''
cleanup() {
    trap - EXIT INT TERM
    for task_pid in "$task_tunnel_pid" "$task_app_pid"; do
        if [[ -n "$task_pid" ]]; then
            kill "$task_pid" 2>/dev/null || true
            wait "$task_pid" 2>/dev/null || true
        fi
    done
    python3 - "$task_runtime" <<'PY'
import shutil, sys
shutil.rmtree(sys.argv[1])
PY
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

go build -o "$task_runtime/app" ./cmd/app
"$task_runtime/app" &
task_app_pid=$!
for ((task_attempt = 0; task_attempt < 100; task_attempt++)); do
    if ! kill -0 "$task_app_pid" 2>/dev/null; then
        wait "$task_app_pid"
        exit 1
    fi
    if curl --fail --silent --max-time 1 "http://127.0.0.1:$SERVER_PORT/healthz" -o /dev/null; then
        break
    fi
    sleep 0.1
done
if ! curl --fail --silent --max-time 1 "http://127.0.0.1:$SERVER_PORT/healthz" -o /dev/null; then
    echo 'Приложение не запустилось за 10 секунд.' >&2
    exit 1
fi

printf 'URL для BotFather: %s\n' "$APP_URL"
if [[ "$task_tunnel_provider" == ngrok ]]; then
    "$task_ngrok" http "http://127.0.0.1:$SERVER_PORT" --url "$APP_URL" --inspect=false &
    task_tunnel_pid=$!
else
    printf 'Используется уже включённый Tailscale Funnel: %s\n' "$APP_URL"
fi
if [[ -z "$task_tunnel_pid" ]]; then
    wait "$task_app_pid"
else
    wait -n "$task_app_pid" "$task_tunnel_pid"
fi

#!/usr/bin/env bash
# Send a short progress message to Aurora's Telegram chat.
#
# Usage: ./scripts/notify.sh "text of the message"
#
# engram has no .env of its own — the token is read from the sibling geodrill
# checkout's .env at send time and never echoed. A failed or impossible send
# never blocks anything (always exits 0).
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
env_file="$repo_root/../../PersonalProjects/geodrill/.env"

if [ ! -f "$env_file" ]; then
  echo "notify: geodrill .env not found — skipping" >&2
  exit 0
fi

set -a
# shellcheck disable=SC1090
source "$env_file"
set +a

# Token and recipient chat id both come from the sibling geodrill .env
# (gitignored) — never from tracked source, since both repos are public.
if [ -z "${TELEGRAM_TOKEN:-}" ]; then
  echo "notify: TELEGRAM_TOKEN not set — skipping" >&2
  exit 0
fi

chat_id="${TELEGRAM_CHAT_ID:-}"
if [ -z "$chat_id" ]; then
  echo "notify: TELEGRAM_CHAT_ID not set in .env — skipping" >&2
  exit 0
fi

curl -s -m 10 -X POST "https://api.telegram.org/bot${TELEGRAM_TOKEN}/sendMessage" \
  --data-urlencode "chat_id=${chat_id}" \
  --data-urlencode "text=${1:-(empty notify)}" \
  -o /dev/null || echo "notify: send failed (non-blocking)" >&2

exit 0

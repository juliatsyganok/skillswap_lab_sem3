#!/bin/sh
# Сценарий A: команда → повтор → чтение и отказы.
# Перед запуском: go run ./cmd/api (перезапуск сбрасывает состояние к фикстурам).
BASE=${BASE:-http://127.0.0.1:8080}
req() { echo "== $*"; curl -s -w ' -> %{http_code}\n' "$@"; }

# Исходное состояние: 200, status=pending
req -H 'X-Demo-User: learner' "$BASE/v1/engagements/response-1"
# Не автор: 403 not_offer_author, состояние не меняется
req -X PUT -H 'X-Demo-User: learner' "$BASE/v1/engagements/response-1/acceptance"
# Подтверждение: 200, status=accepted, changed=true
req -X PUT -H 'X-Demo-User: author' "$BASE/v1/engagements/response-1/acceptance"
# Повтор: 200, status=accepted, changed=false
req -X PUT -H 'X-Demo-User: author' "$BASE/v1/engagements/response-1/acceptance"
# Чтение второй стороной: 200, status=accepted
req -H 'X-Demo-User: learner' "$BASE/v1/engagements/response-1"
# Не участник: 403 not_participant
req -H 'X-Demo-User: mentor' "$BASE/v1/engagements/response-1"
# Нет договорённости: 404 engagement_not_found
req -H 'X-Demo-User: author' "$BASE/v1/engagements/response-404"
# Неверный ID / query / тело / identity: 400
req -H 'X-Demo-User: author' "$BASE/v1/engagements/-bad"
req -X PUT -H 'X-Demo-User: author' "$BASE/v1/engagements/response-1/acceptance?status=accepted"
req -X PUT -H 'X-Demo-User: author' -d '{"actor":"author"}' "$BASE/v1/engagements/response-1/acceptance"
req -X PUT "$BASE/v1/engagements/response-1/acceptance"

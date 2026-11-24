#!/usr/bin/env bash
set -euo pipefail

base="${BASE_URL:-http://127.0.0.1:8080}/v1/agreements"
: "${SKILLSWAP_DATABASE_URL:?Set SKILLSWAP_DATABASE_URL to the demo database}"
body=$(mktemp)
trap 'rm -f "$body"' EXIT

request() {
  local expected_status=$1 expected_body=$2 method=$3 actor=$4 path=$5 status
  shift 5
  status=$(curl -sS --max-time 5 -o "$body" -w '%{http_code}' -X "$method" \
    -H "X-Demo-User: $actor" -H 'Content-Type: application/json' "$base$path" "$@")
  cat "$body"
  if [[ $status != "$expected_status" ]] || ! grep -Fq -- "$expected_body" "$body"; then
    printf 'Expected HTTP %s and %s, got HTTP %s\n' "$expected_status" "$expected_body" "$status" >&2
    exit 1
  fi
}

payload='{"summary":"Skill exchange complete","rating":5}'
request 403 '"code":"forbidden"' POST mentor /agreement-1/completion -d "$payload"
request 404 '"code":"not_found"' POST author /missing/completion -d "$payload"
request 400 '"code":"invalid_request"' POST author /agreement-1/completion -d '{"summary":"done","rating":6}'
request 200 '"changed":true' POST author /agreement-1/completion -d "$payload"
request 200 '"changed":false' POST learner /agreement-1/completion -d '{"summary":"Other result","rating":1}'
request 200 '"completion":{"summary":"Skill exchange complete","rating":5}' GET learner /agreement-1
request 409 '"code":"invalid_transition"' POST author /agreement-1/transfers \
  -d '{"command_id":"demo-transfer","proposal_id":"proposal-1","expected_revision":1}'

state=$(psql "$SKILLSWAP_DATABASE_URL" -At -v ON_ERROR_STOP=1 -c "
  SELECT a.status, a.revision, a.result_summary, a.result_rating, count(r.slot_id)
  FROM agreements a LEFT JOIN reservations r ON r.agreement_id = a.id
  WHERE a.id = 'agreement-1' GROUP BY a.id")
test "$state" = 'completed|2|Skill exchange complete|5|0'
printf 'PostgreSQL: %s\n' "$state"

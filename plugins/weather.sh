#!/usr/bin/env bash
# Weather instant-answer — example of an EXEC plugin (external script).
#
# @keywords: weather, forecast, wttr
# @timeout: 6s
#
# Backend: scripts with an interpreter extension (.sh/.py/.rb/.js/.pl) or the
# ".plugin" marker run as subprocesses. The host writes one JSON request to
# stdin per hook; the script writes one JSON reply to stdout.
#
#   pre_search   -> {"allow":true|false}
#   on_result    -> {"keep":true|false,"result":{...}}
#   post_search  -> {"results":[ {<result map>}, ... ]}
#
# This keeps the gosearx binary pure-Go; interpreters are the user's and the
# plugin is entirely optional. Requires `curl`.
#
# Usage: "weather Tokyo"  /  "forecast Berlin"

set -euo pipefail

payload="$(cat)"

# Only act on post_search.
case "$payload" in
  *'"hook":"post_search"'*) ;;
  *) echo '{}'; exit 0 ;;
esac

# Extract the query string from the JSON (no jq dependency).
query="$(printf '%s' "$payload" | sed -n 's/.*"query":"\([^"]*\)".*/\1/p')"
# Drop the leading keyword (weather/forecast/wttr).
loc="$(printf '%s' "$query" | sed -E 's/^[[:alpha:]]+[[:space:]]+//')"

if [ -z "$loc" ]; then
  echo '{}'
  exit 0
fi

if ! command -v curl >/dev/null 2>&1; then
  echo '{}'
  exit 0
fi

# wttr.in returns a one-line summary with format=4.
enc="$(printf '%s' "$loc" | sed 's/ /+/g')"
summary="$(curl -fsS --max-time 5 "https://wttr.in/${enc}?format=4" 2>/dev/null || true)"

if [ -z "$summary" ]; then
  echo '{}'
  exit 0
fi

# Emit a JSON answer. Escape backslashes and double-quotes for JSON safety.
esc="$(printf '%s' "$summary" | sed 's/\\/\\\\/g; s/"/\\"/g')"
printf '{"results":[{"type":"answer","answer":"%s","url":"https://wttr.in/%s"}]}' "$esc" "$enc"

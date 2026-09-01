#!/usr/bin/env bash
set -euo pipefail

stage=${1:?stage name required}
output=${2:?metrics output required}
shift 2
mkdir -p "$(dirname "$output")"
time_file=$(mktemp)
start_ms=$(date +%s%3N)
set +e
/usr/bin/time -v "$@" 2> >(tee "$time_file" >&2)
status=$?
set -e
end_ms=$(date +%s%3N)
wall_ms=$((end_ms - start_ms))
peak_rss_kib=$(awk -F: '/Maximum resident set size/ {gsub(/^[[:space:]]+/, "", $2); print $2; exit}' "$time_file")
if [[ -z "$peak_rss_kib" ]]; then
  peak_rss_kib=0
fi
jq -n --arg stage "$stage" --argjson wall_ms "$wall_ms" --argjson peak_rss_kib "$peak_rss_kib" --argjson status "$status" '{stage:$stage,wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib,status:$status}' > "$output"
rm -f "$time_file"
exit "$status"

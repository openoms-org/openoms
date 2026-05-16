#!/usr/bin/env bash
set -euo pipefail

root="${1:-apps/api-server/migrations}"
missing=0
orders_write_regex='(UPDATE|DELETE[[:space:]]+FROM|ALTER[[:space:]]+TABLE)[[:space:]]+(public\.)?orders'
orders_index_regex='CREATE[[:space:]]+(UNIQUE[[:space:]]+)?INDEX[^;]+ON[[:space:]]+(public\.)?orders'
orders_drop_index_regex='DROP[[:space:]]+INDEX[^;]+idx_orders'

while IFS= read -r -d '' file; do
  base="$(basename "$file")"

  # 000001 is the clean baseline schema and intentionally creates base indexes.
  if [[ "$base" == 000001_* ]]; then
    continue
  fi

  normalized="$(tr '\n' ' ' < "$file" | tr '\t' ' ')"
  if [[ "$normalized" =~ $orders_write_regex ]] ||
     [[ "$normalized" =~ $orders_index_regex ]] ||
     [[ "$normalized" =~ $orders_drop_index_regex ]]; then
    if ! grep -Eiq "SET[[:space:]]+LOCAL[[:space:]]+lock_timeout" "$file"; then
      echo "Missing SET LOCAL lock_timeout in $file"
      missing=1
    fi
    if ! grep -Eiq "SET[[:space:]]+LOCAL[[:space:]]+statement_timeout" "$file"; then
      echo "Missing SET LOCAL statement_timeout in $file"
      missing=1
    fi
  fi
done < <(find "$root" -type f -name '*.up.sql' -print0 | sort -z)

if [[ "$missing" -ne 0 ]]; then
  echo "High-risk orders migrations must set transaction-local lock and statement timeouts."
  exit 1
fi

echo "Migration timeout guard passed"

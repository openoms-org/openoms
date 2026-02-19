#!/bin/sh
set -e

# Replace build-time placeholder with runtime NEXT_PUBLIC_API_URL.
# Used by docker-compose / local dev. In Kubernetes, the initContainer handles this.
if [ -n "$NEXT_PUBLIC_API_URL" ] && [ "$NEXT_PUBLIC_API_URL" != "NEXT_PUBLIC_API_URL_PLACEHOLDER" ]; then
  find /app/.next/static /app/server.js -type f -name '*.js' 2>/dev/null | while read -r file; do
    if grep -q 'NEXT_PUBLIC_API_URL_PLACEHOLDER' "$file" 2>/dev/null; then
      sed -i "s|NEXT_PUBLIC_API_URL_PLACEHOLDER|${NEXT_PUBLIC_API_URL}|g" "$file"
    fi
  done
fi

exec "$@"

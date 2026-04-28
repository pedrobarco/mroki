#!/bin/sh
set -e

# Generate runtime configuration from environment variables.
# This overwrites the default empty config.js with actual values.
cat <<EOF > /srv/config.js
window.__MROKI__ = {
  API_BASE_URL: "${MROKI_API_BASE_URL}",
  API_KEY: "${MROKI_API_KEY}"
};
EOF

exec "$@"

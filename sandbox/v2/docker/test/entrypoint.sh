#!/bin/bash
# V2 test entrypoint — starts test services then delegates to base entrypoint

python3 /opt/test/ws-echo.py &
python3 /opt/test/sse-server.py &

exec /entrypoint.sh "$@"

#!/usr/bin/env sh
# simple wait-for script (POSIX sh)
# usage: wait-for.sh host:port host2:port2 -- command arg1 arg2 ...

set -eu

# gather dep targets until '--'
targets=()
while [ $# -gt 0 ]; do
  case "$1" in
    --) shift; break ;;
    *) targets+=("$1"); shift ;;
  esac
done

# default timeout (seconds)
TIMEOUT=${WAIT_TIMEOUT:-60}
SLEEP=${WAIT_SLEEP:-1}

echo "Waiting for ${#targets[@]} dependencies (timeout ${TIMEOUT}s)..."
start=$(date +%s)

check() {
  hostport="$1"
  host=$(echo "$hostport" | cut -d: -f1)
  port=$(echo "$hostport" | cut -d: -f2)
  # try to open TCP connection
  nc -z "$host" "$port" >/dev/null 2>&1
  return $?
}

for t in "${targets[@]}"; do
  echo -n " -> $t ... "
  until check "$t"; do
    now=$(date +%s)
    elapsed=$((now - start))
    if [ "$elapsed" -ge "$TIMEOUT" ]; then
      echo "timeout waiting for $t"
      exit 1
    fi
    sleep "$SLEEP"
  done
  echo "ok"
done

# run command
if [ $# -gt 0 ]; then
  echo "All dependencies ready — exec: $*"
  exec "$@"
else
  echo "No command supplied."
  exit 0
fi

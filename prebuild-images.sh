#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"

CTOOL=$(type -p docker || type -p podman)
if [[ -z "$CTOOL" ]]; then
    >&2 echo "no docker/podman tool found"
    exit 1
fi


>&2 echo "=== Building 'scanner'"

(
    set -x
    "$CTOOL" build --target scanner -t censys-takehome-scanner $@  .
)

>&2 echo "=== Building 'processor'"
(
    set -x
    "$CTOOL" build --target processor -t censys-takehome-processor $@  .
)

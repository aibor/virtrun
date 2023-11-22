#!/bin/bash

set -eEuo pipefail

rundir="$(dirname "${BASH_SOURCE[0]}")"

export KERNEL="$($rundir/fetch_kernel.sh)"

exec $@

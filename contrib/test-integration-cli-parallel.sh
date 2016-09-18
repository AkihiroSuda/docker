#!/bin/bash
### Usage:
###     $ sudo apt install parallel
###     $ ./contrib/test-integration-cli-parallel.sh
set -e

: ${NJOBS=$(nproc)}
D=$(pwd)/bundles-parallel
RUNNER=$(pwd)/contrib/.test-integration-cli-parallel
INPUT=$(pwd)/contrib/.test-integration-cli-parallel-commands
PARALLEL=parallel

echo "Running tests in parallel. see $D for results."
set -x
rm -rf $D/results $D/joblog
mkdir -p $D
$PARALLEL \
    --jobs $NJOBS \
    --results $D/results \
    --joblog  $D/joblog \
    --arg-file $INPUT \
    $RUNNER "{#}" "{}"
set +x

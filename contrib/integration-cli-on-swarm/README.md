# Integration Testing on Swarm

IT on Swarm allows you to execute integration test in parallel across a Docker Swarm cluster

## Architecture

### Master service

  - Works as a funker caller
  - Calls a worker funker (`WORKER_SERVICE`) with a test chunk
  - The list of the tests is passed as a file (`INPUT`), and is chunked according to `CHUNKS`. The chunk sizes are determined randomly because the makespans are non-uniform. In future, we could use some statistics to optimize the makespan, rather than relying on random chunking.
  - `INPUT` is provided via `docker secret`

### Worker service

  - Works as a funker callee
  - Executes an equivalent of `TESTFLAGS=-check.f TestFoo -check.f TestBar -check.f TestBaz ... make test-integration-cli` using the bind-mounted API socket (`docker.sock`)

### Client

  - Controls master and workers
  - No need to have a local daemon

Typically, the master/worker services are supposed to be running on a cloud environment,
while the client is supposed to be running on a laptop, e.g. Docker for Mac/Windows.

## Requirement

  - Docker daemon 1.13 with `--experimental` flag
  - Private registry for distributed execution with multiple nodes

## Usage

Prepare the base image:

    $ make build

Execute tests:

    $ ./contrib/integration-cli-on-swarm/integration-cli-on-swarm.sh --replicas 40 --push-worker-image YOUR_REGISTRY.EXAMPLE.COM/integration-cli-worker:latest 


### Flags

* `--replicas N`: the number of worker service replicas. i.e. degree of parallelism.
* `--chunks N`: hint for the number of chunks. By default, `chunks` == `replicas`.
* `--push-worker-image REGISTRY/IMAGE:TAG`: push the worker image to the registry. Note that if you have only single node and hence you do not need a private registry, you do not need to specify `--push-worker-image`.
* `--rand-seed N`: the random seed for chunking. This flag is useful for deterministic replaying. By default, the timestamp is used.
* `--filters-file FILE`: the file contains `-check.f` strings. By default, the file is automatically generated.
* `--dry-run`: skip the actual workload

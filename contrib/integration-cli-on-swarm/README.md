# Integration Testing on Swarm

IT on Swarm allows you to execute integration test in parallel across a Docker Swarm cluster

## Architecture

### Master service

  - Works as a funker caller
  - Calls a worker funker (`WORKER_SERVICE`) with a test chunk
  - The list of the tests is passed as a file (`INPUT`), and is chunked according to `BATCH_SIZE`
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

    $ ./contrib/integration-cli-on-swarm/integration-cli-on-swarm.sh --replicas 40 --batch-size 10 --shuffle --push-worker-image YOUR_REGISTRY.EXAMPLE.COM/integration-cli-worker:latest 


### Flags

* `--replicas N`: specify the number of worker service replicas.
* `--batch-size N`: specify the number of `-check.f` strings passed at once. For better parallelism, you should set the value to lower, but it also increases the overhead.
* `--shuffle`: shuffle the list of `-check.f` strings so as to try to equalize the makespan
* `--push-worker-image REGISTRY/IMAGE:TAG`: push the worker image to the registry. Note that if you have only single node and hence you do not need a private registry, you do not need to specify `--push-worker-image`.
* `--dry-run`: skip the actual workload

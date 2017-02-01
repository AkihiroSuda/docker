# Integration Testing on Swarm

IT on Swarm allows you to execute integration test in parallel across a Docker Swarm cluster

## Architecture

### Master service

  - Works as a funker caller
  - Calls a worker funker (`-worker-service`) with a chunk of `-check.f` filter strings (passed as a file via `-input` flag, typically `/mnt/input`)

### Worker service

  - Works as a funker callee
  - Executes an equivalent of `TESTFLAGS=-check.f TestFoo|TestBar|TestBaz ... make test-integration-cli` using the bind-mounted API socket (`docker.sock`)

### Client

  - Controls master and workers via `docker stack`
  - No need to have a local daemon

Typically, the master and workers are supposed to be running on a cloud environment,
while the client is supposed to be running on a laptop, e.g. Docker for Mac/Windows.

## Requirement

  - Docker daemon 1.13
  - Private registry for distributed execution with multiple nodes

## Usage

Prepare the base image:

    $ make build

Execute tests:

    $ ./contrib/integration-cli-on-swarm/integration-cli-on-swarm.sh --replicas 40 --push-worker-image YOUR_REGISTRY.EXAMPLE.COM/integration-cli-worker:latest 


### Flags

Basic flags:

* `--replicas N`: the number of worker service replicas. i.e. degree of parallelism.
* `--chunks N`: the number of chunks. By default, `chunks` == `replicas`.
* `--push-worker-image REGISTRY/IMAGE:TAG`: push the worker image to the registry. Note that if you have only single node and hence you do not need a private registry, you do not need to specify `--push-worker-image`.

Experimental flags for mitigating makespan nonuniformity

* `--shuffle`: Shuffle the test filter strings

Flags for debugging IT on Swarm itself

* `--rand-seed N`: the random seed. This flag is useful for deterministic replaying. By default(0), the timestamp is used.
* `--filters-file FILE`: the file contains `-check.f` strings. By default, the file is automatically generated.
* `--dry-run`: skip the actual workload

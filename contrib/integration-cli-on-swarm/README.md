# Integration Testing on Swarm

IT on Swarm allows you to execute integration test in parallel across a Docker Swarm cluster

## Architecture

### Master

  - Started via `docker run`
  - Works as a funker caller
  - Calls a worker funker with a test defined in `master/config.yaml`

### Worker 

  - Started via `docker service`
  - Works as a funker callee
  - Executes an equivalent of `TESTFLAGS=-check.f FooBar make test-integration-cli` using the bind-mounted API socket (`docker.sock`)

### Client

  - Controls master and workers
  - No need to have local daemon

Typically, the master and the workers are supposed to be running on some cloud environment,
while the client is supposed to be running on a laptop, e.g. Docker for Mac/Windows.

## Requirement

  - Docker daemon 1.13 with `--experimental` flag
  - Private registry for distributed execution with multiple nodes

## Usage

Prepare the base image:

    $ make build


(Optional) Configure the list of tests to run:

    $ vi ./contrib/integration-cli-on-swarm/master/config.yaml

Execute tests:

    $ ./contrib/integration-cli-on-swarm/integration-cli-on-swarm.sh --replicas 10 --push-worker-image YOUR_REGISTRY.EXAMPLE.COM/integration-cli-worker:latest
	
Note: if you have only single node and hence you do not need a private registry, you do not need to specify `--push-worker-image`.

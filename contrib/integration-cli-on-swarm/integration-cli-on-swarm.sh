#!/bin/bash
# integration-cli-on-swarm.sh: run the integration tests in parall across a Swarm cluster
#
# Architecture:
#  - master: works as a funker caller. Started via `docker run`.
#  - worker: works as a funker callee. Started via `docker service`.
#  Client (e.g. d4m) is not supposed to be a member of the Swarm cluster.
#
# Requirement:
#  - Docker daemon 1.13 with `--experimental` flag
#  - Private registry for distributed execution with multiple nodes
set -e
set -o pipefail


errexit() {
    echo "$1"
    exit 1
}

log(){
    echo -e "\e[104m\e[97m[IT on Swarm]\e[49m\e[39m $@"
}

zleep() {
    sleep 3
}

cleanup() {
    log "Cleaning up..."
    network="$1" master_service="$2" worker_service="$3" secret="$4"
    set -x
    docker service rm $master_service $worker_service && zleep
    docker network rm $network && zleep
    docker secret rm $secret
    set +x
}

build_master_image() {
    name="$1"
    log "Building master image $name"
    set -x
    ( cd contrib/integration-cli-on-swarm/agent; docker image build --tag $name --file Dockerfile.master .)
    set +x
}

build_worker_image() {
    name="$1"
    base="$(make echo-docker-image)"
    # tmp is used as FROM in worker/Dockerfile
    tmp="docker-dev:integration-cli-worker-base"
    log "Building worker image $name from $base"
    log "NOTE: you may need to run \`make build\` for updating $base"
    set -x
    docker image tag $base $tmp
    ( cd contrib/integration-cli-on-swarm/agent; docker image build --tag $name --file Dockerfile.worker .)
    docker image rm --force $tmp
    set +x
}


push_worker_image() {
    name="$1"
    log "Pushing master image $name"
    set -x
    docker image push $name
    set +x
}

create_network(){
    name="$1"
    log "Creating network $name"
    set -x
    docker network create --attachable --driver overlay $name
    set +x
}

create_worker_service(){
    replicas="$1" network="$2" name="$3" image="$4" dry_run="$5"
    # we need the image ID rather than name (#29582)
    image_id=$(docker inspect -f '{{.Id}}' $image)
    log "Creating worker service $name ($replicas replicas, image id=$image_id)"
    set -x
    docker service create \
	   --replicas $replicas \
	   --network $network  \
	   --restart-condition any \
	   --mount type=bind,src=/var/run/docker.sock,target=/var/run/docker.sock \
	   --with-registry-auth \
	   --env "WORKER_IMAGE=$image_id" \
	   --env "DRY_RUN=$dry_run" \
	   --name $name \
	   $image
    set +x
}

create_master_service(){
    batch_size="$1" network="$2" worker="$3" name="$4" image="$5" secret="$6"
    self_node=$(docker node inspect -f '{{.ID}}' self)
    log "Running master container $name (batch size=${batch_size}) on node $self_node"
    set -x
    docker service create \
	   --network $network \
	   --secret $secret \
	   --env WORKER_SERVICE=$worker \
	   --env BATCH_SIZE=$batch_size \
	   --env INPUT=/run/secrets/$secret \
	   --restart-condition none \
	   --constraint "node.id == $self_node" \
	   --name $name $image
    set +x
}

enum_filter_strings(){
    # TODO: refine the command for better maintainability.
    #       Note that we could use `TESTFLAGS=-check.list make test-integration-cli`, but it is slow.
    grep -oPh '^func \(.*\*\K\w+Suite\) Test\w+' integration-cli/*_test.go | sed -e 's/) /./g' | sort
    # The output will be as follows:
    #  DockerAuthzSuite.TestAuthZPluginAPIDenyResponse
    #  DockerAuthzSuite.TestAuthZPluginAllowEventStream
    #  ...
    #  DockerTrustedSwarmSuite.TestTrustedServiceUpdate
}

create_secret() {
    secret="$1"
    log "Saving the list of test as a secret $secret"
    set -x
    enum_filter_strings | docker secret create -f - $secret
    set +x
}

wait_for_master_completion(){
    name="$1"
    container=$(docker inspect -f '{{.Status.ContainerStatus.ContainerID}}' $(docker service ps -q $name) )
    # this works because container is guaranteed to running on the "self" node, using node-constraint
    docker wait $container
}

main() {
    replicas="1"
    batch_size="10"
    master_image="integration-cli-master"
    master_service="integration-cli-master"
    push_worker_image=
    worker_image="integration-cli-worker"
    worker_service="integration-cli-worker"
    network="integration-cli-network"
    secret="integration-cli-secret"
    dry_run=
    while [ "$#" -gt 0 ]; do
	case "$1" in
	    --replicas)
		replicas="$2"
		shift 2
		;;
	    --batch-size)
		batch_size="$2"
		shift 2
		;;
	    --push-worker-image)
		push_worker_image="1"
		worker_image="$2"
		shift 2
		;;
	    --dry-run)
		dry_run="1"
		shift 1
		;;
	    *)
		errexit "Usage: $0 --replicas N --batch-size M --push-worker-image NAME --dry-run"
		;;
	esac
    done

    # Clean up, for just in case
    cleanup $network $master_service $worker_service $secret || true
    zleep

    # Build images
    build_master_image $master_image
    build_worker_image $worker_image
    [ $push_worker_image ] && push_worker_image $worker_image

    # Create network
    create_network $network

    # Create secret
    create_secret $secret

    # Start the services
    zleep # wait for network
    create_worker_service $replicas $network $worker_service $worker_image $dry_run
    create_master_service $batch_size $network $worker_service $master_service $master_image $secret

    # Print service logs in background
    docker service logs --follow $worker_service &
    worker_service_logs_pid=$!
    docker service logs --follow $master_service &
    master_service_logs_pid=$!

    # Register cleaner for ^C
    trap "kill -9 $worker_service_logs_pid > /dev/null 2>&1; cleanup $network $master_service $worker_service $secret || true" INT

    # Wait for master completion
    zleep; zleep # wait so that `docker service inspect` (called from `wait_for_master_completion` works)
    code=$(wait_for_master_completion $master_service)
    cleanup $network $master_service $worker_service $secret || true
    kill -9 $master_service_logs $worker_service_logs_pid  > /dev/null 2>&1
    log "Exit status: $code"
    exit $code
}

main "$@"

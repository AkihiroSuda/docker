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
    network="$1" master_container="$2" worker_service="$3"
    set -x
    docker container rm -f $master_container && zleep
    docker service rm $worker_service && zleep
    docker network rm $network
    set +x
}

build_master_image() {
    name="$1"
    log "Building master image $name"
    set -x
    docker image build --tag $name contrib/integration-cli-on-swarm/master
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
    docker image build --tag $name contrib/integration-cli-on-swarm/worker
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
    replicas="$1" network="$2" name="$3" image="$4"
    log "Creating worker service $name ($replicas replicas)"
    set -x
    docker service create \
	   --replicas $replicas \
	   --network $network  \
	   --restart-condition any \
	   --mount type=bind,src=/var/run/docker.sock,target=/var/run/docker.sock \
	   --with-registry-auth \
	   --env "WORKER_IMAGE=$image" \
	   --name $name \
	   $image
    set +x
}

create_master_container(){
    network="$1" worker="$2" name="$3" image="$4"
    log "Creating master container $name"
    set -x
    docker container run --detach --network $network --env WORKER_SERVICE=$worker --name $name $image
    set +x
}

main() {
    replicas="1"
    master_image="integration-cli-master"
    master_container="integration-cli-master"
    push_worker_image=
    worker_image="integration-cli-worker"
    worker_service="integration-cli-worker" # FIXME: also defined in master/main.go
    network="integration-cli-network"
    while [ "$#" -gt 0 ]; do
	case "$1" in
	    --replicas)
		replicas="$2"
		shift 2
		;;
	    --push-worker-image)
		push_worker_image="1"
		worker_image="$2"
		shift 2
		;;
	    *)
		errexit "Usage: $0 --replicas N --push-worker-image NAME"
		;;
	esac
    done

    # Clean up, for just in case
    cleanup $network $master_container $worker_service || true
    zleep

    # Build images
    build_master_image $master_image
    build_worker_image $worker_image
    [ $push_worker_image ] && push_worker_image $worker_image

    # Start containers
    create_network $network
    create_worker_service $replicas $network $worker_service $worker_image
    zleep # for waiting network is ready
    create_master_container $network $worker_service $master_container $master_image

    # Follow logs
    docker service logs --follow $worker_service &
    worker_service_logs_pid=$!
    docker logs --follow $master_container &
    master_container_logs_pid=$!

    # Wait for completion and clean up
    code=$(docker wait $master_container)
    cleanup $network $master_container $worker_service || true
    kill -9 $worker_service_logs_pid $master_container_logs_pid > /dev/null 2>&1
    log "Exit status: $code"
    exit $code
}

main "$@"

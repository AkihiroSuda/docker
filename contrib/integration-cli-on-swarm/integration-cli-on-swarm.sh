#!/bin/bash
# integration-cli-on-swarm.sh: run the integration tests in parall across a Swarm cluster
# Please refer to README.md for the usage.
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
    network="$1" master_container="$2" worker_service="$3" volume="$4"
    set -x
    docker container rm -f $master_container
    docker service rm $worker_service && zleep
    docker network rm $network && zleep && zleep
    docker volume rm $volume
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
    log "Pushing worker image $name"
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

enum_filter_strings(){
    # TODO: refine the command for better maintainability.
    #       Note that we could use `TESTFLAGS=-check.list make test-integration-cli`, but it is slow.
    grep -oPh '^func \(.*\*\K\w+Suite\) Test\w+' integration-cli/*_test.go | sed -e 's/) /./g' | sed -e 's/$/\$/g' | sort
    # The output will be as follows:
    #  DockerAuthzSuite.TestAuthZPluginAPIDenyResponse$
    #  DockerAuthzSuite.TestAuthZPluginAllowEventStream$
    #  ...
    #  DockerTrustedSwarmSuite.TestTrustedServiceUpdate$
}

create_volume() {
    volume="$1"
    log "Creating volume $volume"
    set -x
    docker volume create --driver local $volume
    set +x
}

create_input() {
    volume="$1" file="$2"
    if [ -z $file ]; then
	file=$(mktemp)
	log "Generating the list of test filter strings as $file"
	enum_filter_strings > $file
    fi
    log "Saving the list of test filter strings ($file) as a volume $volume"
    set -x
    cat $file | docker run -i --rm -v $volume:/mnt busybox sh -c "cat > /mnt/input"
    set +x
}

create_worker_service(){
    # TODO: we should better use bash4 dictionary object?
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
	   --name $name \
	   $image \
	   -worker-image=$image_id \
	   -dry-run=$dry_run
    set +x
}

run_master_container(){
    # TODO: we should better use bash4 dictionary object?
    chunks="$1" network="$2" worker="$3" name="$4" image="$5" volume="$6" shuffle="$7" rand_seed="$8"
    self_node=$(docker node inspect -f '{{.ID}}' self)
    log "Running master container $name on node"
    set -x
    docker container run -it --rm \
	   --network $network \
	   -v $volume:/mnt \
	   --name $name \
	   $image \
	   -worker-service=$worker \
	   -chunks=$chunks \
	   -input=/mnt/input \
	   -shuffle=$shuffle \
	   -rand-seed=$rand_seed
    code=$?
    set +x
    return $code
}

main() {
    network="integration-cli-network"
    volume="integration-cli-volume"
    master_image="integration-cli-master"
    master_container="integration-cli-master"
    worker_image="integration-cli-worker"
    worker_service="integration-cli-worker"
    replicas="1"
    # empty denotes $replicas
    chunks=
    # empty denotes not to push
    push_worker_image=
    shuffle="false"
    # zero denotes timestamp
    rand_seed=0
    # empty denotes to generate the file automatically
    filters_file=
    dry_run="false"
    while [ "$#" -gt 0 ]; do
	case "$1" in
	    --replicas)
		replicas="$2"
		shift 2
		;;
	    --chunks)
		chunks="$2"
		shift 2
		;;
	    --push-worker-image)
		push_worker_image="1"
		worker_image="$2"
		shift 2
		;;
	    --shuffle)
		shuffle="true"
		shift 1
		;;
	    --rand-seed)
		rand_seed="$2"
		shift 2
		;;
	    --filters-file)
		filters_file="$2"
		shift 2
		;;
	    --dry-run)
		dry_run="true"
		shift 1
		;;
	    *)
		errexit "Usage: $0 --replicas N --chunks N --push-worker-image NAME --shuffle --rand-seed N --filters-file NAME --dry-run"
		;;
	esac
    done
    [ -z $chunks ] && chunks=$replicas

    # Clean up, for just in case
    cleanup $network $master_container $worker_service $volume || true
    zleep

    # Build images
    build_master_image $master_image
    build_worker_image $worker_image
    [ $push_worker_image ] && push_worker_image $worker_image

    # Create network and volume
    create_network $network
    create_volume $volume

    # Create the list of test filter strings
    create_input $volume $filters_file

    # Start the workers
    zleep # wait for network
    create_worker_service $replicas $network $worker_service $worker_image $dry_run

    # Print service logs in background (FIXME: not printed sometimes?)
    docker service logs --follow $worker_service &
    worker_service_logs_pid=$!

    # Start the master
    set +e
    run_master_container $chunks $network $worker_service $master_container $master_image $volume $shuffle $rand_seed

    # Wait for master completion
    code=$?
    kill -9 $worker_service_logs_pid
    cleanup $network $master_container $worker_service $volume || true
    log "Exit status: $code"
    exit $code
}

main "$@"

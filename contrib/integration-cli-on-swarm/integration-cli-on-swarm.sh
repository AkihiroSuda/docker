#!/bin/bash
# integration-cli-on-swarm.sh: run the integration tests in parall across a Swarm cluster
# Please refer to README.md for the usage.
#
# TODO(AkihiroSuda): rewrite in Go? (Maybe it would just result in increase of LOC?)
set -e
set -o pipefail

# global constants
stack="integration-cli-on-swarm"
volume="integration-cli-on-swarm"
master_image="integration-cli-master"
worker_image="integration-cli-worker"
compose_file="./contrib/integration-cli-on-swarm/docker-compose.yml"

log(){
    echo -e "\e[104m\e[97m[IT on Swarm]\e[49m\e[39m $@"
}

cleanup_stack() {
    [ $# -eq 0 ]
    if docker stack ls | grep $stack > /dev/null; then
	log "Cleaning up stack $stack"
	set -x
	docker stack rm $stack
	# FIXME: make sure all resources are removed here
        sleep 10
	set +x
    fi
}

cleanup_volume() {
    [ $# -eq 0 ]
    if docker inspect $volume > /dev/null 2>&1; then
	log "Cleaning up volume $volume"
	set -x
	docker volume rm $volume
	set +x
    fi
}

ensure_images() {
    [ $# -eq 1 ]; push_worker_image="$1"
    log "Checking $master_image and $worker_image exists"
    # We do not need to always build them. A user may want to run integration-cli-on-swarm.sh multiple times with the same image.
    docker image inspect $master_image $worker_image > /dev/null || log "Please run \`make build-integration-cli-on-swarm\` first"
    if [ $push_worker_image ]; then
	log "Pushing $push_worker_image"
	set -x
	# TODO: skip pushing if no change since last push
	docker image tag $worker_image $push_worker_image
	docker image push $push_worker_image
	set +x
    fi
}

enum_filter_strings(){
    [ $# -eq 0 ]
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
    [ $# -eq 0 ]
    log "Creating volume $volume"
    set -x
    docker volume create --driver local $volume
    set +x
}

create_input() {
    [ $# -le 1 ]; file="$1"
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

create_compose_file() {
    # TODO: use bash4 hash map (NOTE: macOS still ships with bash3)
    [ $# -eq 6 ]; push_worker_image="$1" replicas="$2" chunks="$3" shuffle="$4" rand_seed="$5" dry_run="$6"
    compose_worker_image=$worker_image
    [ $push_worker_image ] && compose_worker_image=$push_worker_image
    worker_image_digest=$(docker inspect -f '{{index .RepoDigests 0}}' $worker_image)
    self_node_id=$(docker node inspect -f '{{.ID}}' self)
    template=$(cat ./contrib/integration-cli-on-swarm/docker-compose.template.yml)
    log "Creating ${compose_file}"
    eval "echo \"${template}\"" > $compose_file
}

create_stack() {
    [ $# -eq 0 ]
    log "Creating stack $stack from ${compose_file}"
    set -x
    docker stack deploy --compose-file ${compose_file} --with-registry-auth $stack
    set +x
}

inspect_master_container_id() {
    [ $# -eq 0 ]
    # FIXME(AkihiroSuda): we should not rely on internal service naming convention
    docker ps --all --quiet \
	   --filter label=com.docker.stack.namespace=${stack} \
	   --filter label=com.docker.swarm.service.name=${stack}_master
}

main() {
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
		push_worker_image="$2"
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
		echo "Usage: $0 --replicas N --chunks N --push-worker-image NAME --shuffle --rand-seed N --filters-file NAME --dry-run"
		exit 1
		;;
	esac
    done
    [ -z $chunks ] && chunks=$replicas

    # Clean up previous experiment
    cleanup_stack
    cleanup_volume

    # Ensure images and push if required
    ensure_images $push_worker_image

    # Create the list of test filter strings
    create_volume
    create_input $filters_file

    # Create the stack
    create_compose_file $push_worker_image $replicas $chunks $shuffle $rand_seed $dry_run
    create_stack
    log "Created stack $stack"
    log "The log will be displayed here after some duration." # worker logs are sent to master in batch
    log "You can watch the live status via \`docker service logs ${stack}_worker\`"

    # Follow the log and wait for the completion
    sleep 10 # FIXME: it should retry until master is up, rather than pre-sleeping
    master_container_id=$(inspect_master_container_id)
    docker container logs --follow $master_container_id

    # Inspect and propagate the exit code
    code=$(docker container inspect --format '{{.State.ExitCode}}' $master_container_id)
    log "Exit status: $code"
    log "NOTE: You may want to inspect or clean up following resources:"
    log " - Volume: $volume" # in future this should contain useful logs (in machine-readable struct maybe)"
    log " - Stack: $stack"   # you should be able to do `docker service logs`
    log "Also, you can clean following resources:"
    log " - Compose file: $compose_file"
    log " - Image (master): $master_image"
    if [ $push_worker_image ]; then
	log " - Image (worker): $worker_image (pushed as $push_worker_image)"
    else
	log " - Image (worker): $worker_image"
    fi
    exit $code
}

main "$@"

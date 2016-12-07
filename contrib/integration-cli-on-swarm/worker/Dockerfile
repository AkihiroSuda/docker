# "docker-dev:integration-cli-worker-base" is automatically generated via integration-cli-on-swarm.sh
FROM docker-dev:integration-cli-worker-base

# Pre-built binaries here for faster worker execution
# TODO: somehow make DOCKER_INCREMENTAL_BINARY effective here for faster compilation
# NOTE: /usr/local/bin/docker is used for communication with the daemon via the bind-mounted API socket
RUN hack/make.sh build-integration-test-binary dynbinary && cp -L  bundles/latest/dynbinary-client/docker /usr/local/bin

ENV FUNKER_GO_COMMIT 37fd8f2b349f311ecd6cd7ca0a1374a56ef6908e
RUN git clone https://github.com/bfirsh/funker-go.git /go/src/github.com/bfirsh/funker-go \
	&& (cd /go/src/github.com/bfirsh/funker-go && git checkout -q $FUNKER_GO_COMMIT)
ADD *.go /
RUN ( cd / && go build -o worker )

# NOTE: WORKDIR is inherited from the base image
ENTRYPOINT /worker

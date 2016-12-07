# "docker-dev:integration-cli-worker-base" is automatically generated via integration-cli-on-swarm.sh
FROM docker-dev:integration-cli-worker-base

# Pre-built binaries here for faster worker execution
# TODO: somehow make DOCKER_INCREMENTAL_BINARY effective here for faster compilation
RUN hack/make.sh build-integration-test-binary dynbinary

ENV FUNKER_GO_COMMIT 8064242118edd1ea555ee2f3d5486ba3414fed06
RUN git clone https://github.com/bfirsh/funker-go.git /go/src/github.com/bfirsh/funker-go \
	&& (cd /go/src/github.com/bfirsh/funker-go && git checkout -q $FUNKER_GO_COMMIT)
	
# docker client is used for communication with the daemon via the bind-mounted API socket
RUN curl -sSL https://master.dockerproject.org/linux/amd64/docker -o /usr/local/bin/docker \
	&& chmod +x /usr/local/bin/docker

ADD *.go /
RUN ( cd / && go build -o worker )

# NOTE: WORKDIR is inherited from the base image
ENTRYPOINT /worker

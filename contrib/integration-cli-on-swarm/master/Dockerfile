FROM golang:1.7

ENV YAML_V2_COMMIT a83829b6f1293c91addabc89d0571c246397bbf4
RUN git clone https://gopkg.in/yaml.v2.git /go/src/gopkg.in/yaml.v2 \
	&& (cd /go/src/gopkg.in/yaml.v2 && git checkout -q $YAML_V2_COMMIT)

ENV FUNKER_GO_COMMIT 8064242118edd1ea555ee2f3d5486ba3414fed06
RUN git clone https://github.com/bfirsh/funker-go.git /go/src/github.com/bfirsh/funker-go \
	&& (cd /go/src/github.com/bfirsh/funker-go && git checkout -q $FUNKER_GO_COMMIT)

WORKDIR /
ADD *.go /
RUN go build -o master
ADD config.yaml /
ENTRYPOINT /master

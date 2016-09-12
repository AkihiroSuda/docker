# [go-fuzz](https://github.com/dvyukov/go-fuzz) for Dockerfile

## Usage

Install go-fuzz (please also refer to https://github.com/dvyukov/go-fuzz):

```console
$ go get github.com/dvyukov/go-fuzz/go-fuzz
$ go get github.com/dvyukov/go-fuzz/go-fuzz-build
```

Build the fuzzer:

```console
$ go-fuzz-build github.com/docker/docker/contrib/fuzz/dockerfile
```

Initialize the corpus:

```console
$ mkdir -p /tmp/fuzz/corpus
$ for f in $(find /$ANYWHERE -name Dockerfile ); do cp $f /tmp/fuzz/corpus/initial-$RANDOM; done
```

Run the fuzzer:

```console
$ go-fuzz -bin=./dockerfile-fuzz.zip -workdir=/tmp/fuzz
```

Wait until it founds a "crasher".

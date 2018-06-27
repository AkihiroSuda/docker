# Rootless mode (Experimental)

The rootless mode allows running `dockerd` as an unprivileged user, using `user_namespaces(7)`, `mount_namespaces(7)`, `network_namespaces(7)`, and [slirp4netns](https://github.com/rootless-containers/slirp4netns).

No SUID binary is required except `newuidmap` and `newgidmap`.

## Requirements
* `newuidmap` and `newgidmap` need to be installed on the host. These commands are provided by the `uidmap` package on most distros.

* `/etc/subuid` and `/etc/subgid` should contain >= 65536 sub-IDs. e.g. `penguin:231072:65536`.

```console
$ id -u
1001
$ whoami
penguin
$ grep ^$(whoami): /etc/subuid
penguin:231072:65536
$ grep ^$(whoami): /etc/subgid
penguin:231072:65536
```

* Some distros such as Debian (excluding Ubuntu) and Arch Linux require `echo 1 > /proc/sys/kernel/unprivileged_userns_clone`.

## Restrictions

* Only `vfs` graphdriver is supported. However, on [Ubuntu](http://kernel.ubuntu.com/git/ubuntu/ubuntu-artful.git/commit/fs/overlayfs?h=Ubuntu-4.13.0-25.29&id=0a414bdc3d01f3b61ed86cfe3ce8b63a9240eba7) and a few distros, `overlay2` and `overlay` are also supported. [Starting with Linux 4.18](https://www.phoronix.com/scan.php?page=news_item&px=Linux-4.18-FUSE), we will be also able to implement FUSE snapshotters.
* Cgroups (including `docker top`) and AppArmor are disabled at the moment. (FIXME: we could enable Cgroups if configured on the host)
* Checkpoint is not supported at the moment.
* Running rootless `dockerd` in rootless/rootful `dockerd` is also possible, but not fully tested.

## Usage

### Daemon
Before running `dockerd` you need to unshare userns, mountns, and netns. Also, sysfs might need to be mounted.

You may use [RootlessKit](https://github.com/rootless-containers/rootlesskit) and [slirp4netns](https://github.com/moby/slirp4netns) for setting up them with usermode NAT.

If your `/etc/resolv.conf` is managed by systemd or NetworkManager, you need to run RootlessKit with `--copy-up=/etc` so as to prevent `/etc/resolv.conf` in the namespace from being unexpectedly unmounted when `/etc/resolv.conf` is recreated on the host.

Also, currently you need to mount `/run/docker` as tmpfs before running `dockerd`, because "/run/docker/libnetwork" is still hard-coded in `vendor/github.com/docker/libnetwork/sandbox_externalkey_unix.go`.

```
$ docker-rootlesskit --net=slirp4netns --slirp4netns-binary=docker-slirp4netns --copy-up=/etc sh -ec "mount -t tmpfs none /run/docker; dockerd --experimental"
```

If `/run/docker` mount point is not available on your host, you can create the mount point by running RootlessKit with `--copy-up=/run`. On some distros, you may also need to hide `/run/xtables.lock` with `--copy-up=run`.

```console
$ docker-rootlesskit --net=slirp4netns --slirp4netns-binary=docker-slirp4netns --copy-up=/etc --copy-up=/run sh -ec "mkdir -p /run/docker; mount -t tmpfs none /run/docker; rm -f /run/xtables.lock; dockerd --experimental"
```

Remarks:
* The socket path is set to `/run/user/$UID/docker.sock` by default.
* The data dir is set to `~/.local/share/docker` by default.
* The exec dir is set to `/run/user/$UID/docker` by default.
* The config dir is set to `~/.config/docker` (not `~/.docker`) by default.

### Client

You can just use the upstream Docker client (without `nsenter`-ing to the `dockerd` namespaces), but you need to set the socket path.

```console
$ docker -H unix:///run/user/1001/docker.sock run -d nginx
```

### Exposing ports

In addition to exposing container ports to the `dockerd` network namespace, you also need to expose the ports in the `dockerd` network namespace to the host network namespace.

e.g.
```console
$ docker-rootlesskit --state=/tmp/foo ...
```

```console
$ docker -H unix:///run/user/1001/docker.sock run -d -p 80:80 nginx
$ socat -t -- TCP-LISTEN:8080,reuseaddr,fork EXEC:"nsenter -U -n -t $(cat /tmp/foo/child_pid) socat -t -- STDIN TCP4\:127.0.0.1\:80"
```

In future, we could integrate RootlessKit into `dockerd` for exposing the `dockerd` namespace ports automatically.

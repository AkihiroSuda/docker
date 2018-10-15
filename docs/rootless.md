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

* Some distros such as Debian (excluding Ubuntu) and Arch Linux require `sudo sh -c "echo 1 > /proc/sys/kernel/unprivileged_userns_clone"`.
* Some distros require `sudo modprobe ip_tables iptable_mangle iptable_nat iptable_filter`.

## Restrictions

* Only `vfs` graphdriver is supported. However, on [Ubuntu](http://kernel.ubuntu.com/git/ubuntu/ubuntu-artful.git/commit/fs/overlayfs?h=Ubuntu-4.13.0-25.29&id=0a414bdc3d01f3b61ed86cfe3ce8b63a9240eba7) and a few distros, `overlay2` and `overlay` are also supported. [Starting with Linux 4.18](https://www.phoronix.com/scan.php?page=news_item&px=Linux-4.18-FUSE), we will be also able to implement FUSE snapshotters.
* Cgroups (including `docker top`) and AppArmor are disabled at the moment. In future, Cgroups will be optionally available when delegation permission is configured on the host.
* Checkpoint is not supported at the moment.
* Running rootless `dockerd` in rootless/rootful `dockerd` is also possible, but not fully tested.

## Usage

### Daemon

You need to run `dockerd-rootless.sh` instead of `dockerd`.

```console
$ dockerd-rootless.sh --experimental"
```
As Rootless mode is experimental per se, currently you always need to run `dockerd-rootless.sh` with `--experimental`.

Remarks:
* The socket path is set to `$XDG_RUNTIME_DIR/docker.sock` by default. `$XDG_RUNTIME_DIR` is typically set to `/run/user/$UID`.
* The data dir is set to `~/.local/share/docker` by default.
* The exec dir is set to `$XDG_RUNTIME_DIR/docker` by default.
* The config dir is set to `~/.config/docker` (not `~/.docker`) by default.
* The `dockerd-rootless.sh` script executes `dockerd` in its own user, mount, and network namespace. You can enter the namespaces by running `nsenter -U --preserve-credentials -n -m -t $(cat $XDG_RUNTIME_DIR/docker.pid)`.

### Client

You can just use the upstream Docker client but you need to set the socket path explicitly.

```console
$ docker -H unix://$XDG_RUNTIME_DIR/docker.sock run -d nginx
```

### Exposing ports

In addition to exposing container ports to the `dockerd` network namespace, you also need to expose the ports in the `dockerd` network namespace to the host network namespace.

```console
$ docker -H unix://$XDG_RUNTIME_DIR/docker.sock run -d -p 80:80 nginx
$ socat -t -- TCP-LISTEN:8080,reuseaddr,fork EXEC:"nsenter -U -n -t $(cat $XDG_RUNTIME_DIR/docker.pid) socat -t -- STDIN TCP4\:127.0.0.1\:80"
```

In future, `dockerd` will be able to expose the ports automatically. See https://github.com/rootless-containers/rootlesskit/issues/14 .

### Routing ping packets

To route ping packets, you need to set up `net.ipv4.ping_group_range` properly as the root.

```console
$ sudo sh -c "echo 0   2147483647  > /proc/sys/net/ipv4/ping_group_range"
```

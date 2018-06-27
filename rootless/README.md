# Rootless mode (Experimental)

The rootless mode allows running `dockerd` as an unprivileged user, using `user_namespaces(7)`.

Requirements:
- `newuidmap` and `newgidmap` need to be installed on the host. These commands are provided by the `uidmap` package.
- `/etc/subuid` and `/etc/subgid` should contain >= 65536 sub-IDs. e.g. `penguin:231072:65536`.
- Some distros such as Debian and Arch Linux require `echo 1 > /proc/sys/kernel/unprivileged_userns_clone`
- To run in a Docker container with non-root `USER` (*UNTESTED*), `docker run --privileged` is still required. See also Jessie's blog: https://blog.jessfraz.com/post/building-container-images-securely-on-kubernetes/
- Currently, because of a libnetwork issue (see below), `/run/docker` directory needs to be pre-created by the root. (No need to change the owner and permissions)

Remarks:

* The socket path will be set to `/run/user/$UID/docker.sock` by default.
* The data dir will be set to `~/.local/share/docker` by default.
* The exec dir will be set to `/run/user/$UID/docker` by default.
* The config dir will be set to `~/.config/docker` (not `~/.docker`) by default.
* `overlay2` graphdriver is not supported except on [Ubuntu-flavored kernel](http://kernel.ubuntu.com/git/ubuntu/ubuntu-artful.git/commit/fs/overlayfs?h=Ubuntu-4.13.0-25.29&id=0a414bdc3d01f3b61ed86cfe3ce8b63a9240eba7). `vfs` graphdriver should work on non-Ubuntu kernel.
* Cgroups, AppArmor, and SELinux are disabled at the moment. (FIXME: we could enable Cgroups if configured on the host)

## Usage

Before running `dockerd` you need to unshare userns, mountns, and netns.

You may use [RootlessKit](https://github.com/AkihiroSuda/rootlesskit) for unsharing them and [VPNKit](https://github.com/moby/vpnkit) for enabling usermode networking.

Also, currently you need to mount `/run/docker` as tmpfs before running `dockerd`, because "/run/docker/libnetwork" is still hard-coded in `vendor/github.com/docker/libnetwork/sandbox_externalkey_unix.go`.

```
$ rootlesskit --net=vpnkit sh -ec "mount -t tmpfs none /run/docker; dockerd --experimental"
```

```
$ docker -H unix:///run/user/1001/docker.sock run -d nginx
```

TODO: build rootlesskit and vpnkit in Dockerfile

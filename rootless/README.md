# Rootless mode (Experimental)

Requirements:
- Some distros such as Debian and Arch Linux require `echo 1 > /proc/sys/kernel/unprivileged_userns_clone`
- `newuidmap` and `newgidmap` need to be installed on the host. These commands are provided by the `uidmap` package.
- `/etc/subuid` and `/etc/subgid` should contain >= 65536 sub-IDs. e.g. `penguin:231072:65536`.
- To run in a Docker container with non-root `USER` (*UNTESTED*), `docker run --privileged` is still required. See also Jessie's blog: https://blog.jessfraz.com/post/building-container-images-securely-on-kubernetes/

Remarks:

* The data dir will be set to `/home/$USER/.local/share/docker` by default.
* The exec dir will be set to `/run/user/$UID/docker` by default.
* `overlay2` graphdriver is not supported except on [Ubuntu-flavored kernel](http://kernel.ubuntu.com/git/ubuntu/ubuntu-artful.git/commit/fs/overlayfs?h=Ubuntu-4.13.0-25.29&id=0a414bdc3d01f3b61ed86cfe3ce8b63a9240eba7). `vfs` graphdriver should work on non-Ubuntu kernel.
* Cgroups, AppArmor, and SELinux are disabled at the moment. (FIXME: we could enable Cgroups if configured on the host)

## Usage

Before running `dockerd` you need to unshare userns, mountns, and netns.

You may use [rootlesskit](https://github.com/AkihiroSuda/rootlesskit) for unsharing them and [VPNKit](https://github.com/moby/vpnkit) for enabling usermode networking.

```
$ rootlesskit --net=vpnkit bash
rootlesskit$ mount -t tmpfs /run/docker # FIXME
rootlesskit$ mount -t tmpfs /etc/docker # FIXME
rootlesskit$ dockerd -H unix:///run/user/1001/docker.sock --experimental
```

```
$ docker -H unix:///run/user/1001/docker.sock run -d --security-opt apparmor=unconfined nginx
```

TODO: build rootlesskit and vpnkit in Dockerfile

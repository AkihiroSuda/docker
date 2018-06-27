#!/bin/sh
set -e -x
if [ -z $XDG_RUNTIME_DIR ]; then
    echo "XDG_RUNTIME_DIR needs to be set"
    exit 1
fi
if [ -z $HOME ]; then
    echo "HOME needs to be set"
    exit 1
fi
ROOTLESSKIT_STATE_DIR=$XDG_RUNTIME_DIR/dockerd-rootless

if [ -z $_DOCKERD_ROOTLESS_CHILD ]; then
    _DOCKERD_ROOTLESS_CHILD=1
    export _DOCKERD_ROOTLESS_CHILD
    # --copy-up allows removing/creating files in the directories by creating tmpfs and symlinks
    # * /etc: copy-up is required so as to prevent `/etc/resolv.conf` in the
    #         namespace from being unexpectedly unmounted when `/etc/resolv.conf` is recreated on the host
    #         (by either systemd-networkd or NetworkManager)
    # * /run: copy-up is required so that we can create /run/docker (hardcoded for plugins) in our namespace
    rootlesskit \
        --state-dir ROOTLESSKIT_STATE_DIR \
        --net=slirp4netns --mtu=65520 \
        --copy-up=/etc --copy-up=/run \
        $0 $@
else
    [ $_DOCKERD_ROOTLESS_CHILD = 1 ]
    rm -f /run/docker /run/xtables.lock
    mkdir -p /run/docker
    mount -t tmpfs none /run/docker
    dockerd $@
fi

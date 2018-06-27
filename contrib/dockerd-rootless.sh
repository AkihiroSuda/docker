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

if [ -z $_DOCKERD_ROOTLESS_CHILD ]; then
    _DOCKERD_ROOTLESS_CHILD=1
    export _DOCKERD_ROOTLESS_CHILD
    # Re-exec the script via RootlessKit, so as to create unprivileged {user,mount,network} namespaces.
    #
    # --net specifies the network stack. slirp4netns, vpnkit, and vdeplug_slirp are supported.
    # Currently, slirp4netns is the fastest.
    # See https://github.com/rootless-containers/rootlesskit for the benchmark result.
    #
    # --copy-up allows removing/creating files in the directories by creating tmpfs and symlinks
    # * /etc: copy-up is required so as to prevent `/etc/resolv.conf` in the
    #         namespace from being unexpectedly unmounted when `/etc/resolv.conf` is recreated on the host
    #         (by either systemd-networkd or NetworkManager)
    # * /run: copy-up is required so that we can create /run/docker (hardcoded for plugins) in our namespace
    rootlesskit \
        --net=slirp4netns --mtu=65520 \
        --copy-up=/etc --copy-up=/run \
        $0 $@
else
    [ $_DOCKERD_ROOTLESS_CHILD = 1 ]
    # remove the symlinks for the existing files in the parent namespace if any,
    # so that we can create our own files in our mount namespace.
    rm -f /run/docker /run/xtables.lock
    dockerd $@
fi

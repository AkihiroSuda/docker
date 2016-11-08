# Docker Experimental Features

This page contains a list of features in the Docker engine which are
experimental. Experimental features are **not** ready for production. They are
provided for test and evaluation in your sandbox environments.

The information below describes each feature and the GitHub pull requests and
issues associated with it. If necessary, links are provided to additional
documentation on an issue.  As an active Docker user and community member,
please feel free to provide any feedback on these features you wish.

## Use Docker experimental

Experimental features are now included in the standard Docker binaries as of
version 1.13.0.
For enabling experimental features, you need to start the Docker daemon with
`--experimental` flag.
You can also enable the daemon flag via `/etc/docker/daemon.json`. e.g.

        {
            "experimental": true
        }

Then make sure the experimental flag is enabled:

        $ docker version -f '{{.Server.Experimental}}'
        true

## Install from `experimental.docker.com`

If you install Docker from `experimental.docker.com`, the experimental features
are enabled by default.
The experimental package available at `experimental.docker.com` is almost
identical to the standard Docker package, but it contains `daemon.json` for
enabling the experimental features by default.

Starting with version 1.13.0, the experimental package is released on the same
schedule as the standard package.
From one release to the next, new features may appear, while existing
experimental features may be refined or entirely removed.

1. Verify that you have `curl` installed.

        $ which curl

    If `curl` isn't installed, install it after updating your manager:

        $ sudo apt-get update
        $ sudo apt-get install curl

2. Get the latest Docker package.

        $ curl -sSL https://experimental.docker.com/ | sh

    The system prompts you for your `sudo` password. Then, it downloads and
    installs Docker and its dependencies.

	>**Note**: If your company is behind a filtering proxy, you may find that the
	>`apt-key`
	>command fails for the Docker repo during installation. To work around this,
	>add the key directly using the following:
	>
	>       $ curl -sSL https://experimental.docker.com/gpg | sudo apt-key add -

3. Verify `docker` is installed correctly.

        $ sudo docker run hello-world

    This command downloads a test image and runs it in a container.

4. And verify the experimental flag is enabled:

        $ docker version -f '{{.Server.Experimental}}'
        true


## Current experimental features

 * [External graphdriver plugins](plugins_graphdriver.md)
 * [Ipvlan Network Drivers](vlan-networks.md)
 * [Docker Stacks and Distributed Application Bundles](docker-stacks-and-bundles.md)
 * [Checkpoint & Restore](checkpoint-restore.md)

## How to comment on an experimental feature

Each feature's documentation includes a list of proposal pull requests or PRs associated with the feature. If you want to comment on or suggest a change to a feature, please add it to the existing feature PR.

Issues or problems with a feature? Inquire for help on the `#docker` IRC channel or in on the [Docker Google group](https://groups.google.com/forum/#!forum/docker-user).

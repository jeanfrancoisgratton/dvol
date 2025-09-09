# dvol

Container volume backup and restore tool
___

## Overview
This tool allows you to back up and restore docker or podman volumes.

This tool relies on the REST APIs which are quite compatible between Docker and Podman. Well... at least enough for the functionalities needed in this tool.

This makes the tool OCR backend agnostic, and as much as possible, API version agnostic.

## Usage

### Backup a volume:
`dvol backup $VOLUME $ARCHIVE`

The volume `VOLUME` will be archived in `ARCHIVE`. If no extension is provided to `ARCHIVE`, `.tar` will be appended.
If either `.tar.gz` or `.tgz` is added, the archive will be gzip-compressed

### Restore a volume:
`dvol restore $VOLUME $ARCHIVE`
Well... the exact opposite to the previous command.

Moreover, on top of .tar and .tar.gz, xz-decompression is also supported.

### List the volumes on the target daemon:
`dvol ls`

### Useful flags
- `-a` : pin the REST API version to use, instead of leaving the API negotiation to the daemons
- `-H` : target a remote docker/podman daemon; this is in the form of `-H host:port` format
- `-i` : docker/podman image to use for the temp container needed in backup
- `-n` : if invoked, there will be no container and image removal once the operations conclude
- `-q` : quiet (minimalist) output
- `-l` : loglevel : supported levels are none (default), error, info and debug.
- `-f` : the fastfail timeout, in seconds. How many seconds for the server handshake to fail; useful with remote daemons
- `-t` : streaming timeout, in minutes. How many minutes of http activity before the connection is considered dead

## Installing...

### Building from source

- Clone the repo
- Switch to the `src/` directory
- Run `./updateBuildDeps.sh`
- Run `./build.sh` (have a look at the script to see offered options)

### Using the binary packages

Under the `Releases` link you should find Alpine (APK), RedHat-based (RPM) and Debian-based (DEB) packages.
*Please note* that The `Releases` link might or might not be present; that automated part of my CI/CD often breaks.

### A note about building packages

The following directories are used in my own CI-CD chain at home:

- `.tito`
- `__debian`
- `__alpine`
- files `rpmbuild-deps.sh` and `dvol.spec`

Eventually, I will publish the artifacts to build the containers that use those files/directories, but as of now, they are way too customized for me to publish.
It's a bummer, those containers work oh-so-well ;)

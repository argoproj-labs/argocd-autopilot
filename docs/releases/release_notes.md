### Changes

- [fix] fix nil pointer deref in provider code when there are network errors [#403](https://github.com/argoproj-labs/argocd-autopilot/pull/403)
- [chore] bumped argo-cd to 2.5.2, removed applicationset pkg (already in argo-cd), updated golang to 1.19 [#394](https://github.com/argoproj-labs/argocd-autopilot/pull/394)
- [fix] add support for git servers with self-signed certificates [#392](https://github.com/argoproj-labs/argocd-autopilot/pull/392)

### Contributors:

- Roi Kramer ([@roi-codefresh](https://github.com/roi-codefresh))

## Installation:

To use the `argocd-autopilot` CLI you need to download the latest binary from the [git release page](https://github.com/argoproj-labs/argocd-autopilot/releases).

### Using brew:

```bash
# install
brew install argocd-autopilot

# check the installation
argocd-autopilot version
```

### Using scoop:

```bash
# update
scoop update

# install
scoop install argocd-autopilot

# check the installation
argocd-autopilot version
```

### Using chocolatey:

```bash
# install
choco install argocd-autopilot

# check the installation
argocd-autopilot version
```

### Linux and WSL (using curl):

```bash
# download and extract the binary
<<<<<<< HEAD
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.9/argocd-autopilot-linux-amd64.tar.gz | tar zx
=======
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.8/argocd-autopilot-linux-amd64.tar.gz | tar zx
>>>>>>> Release-v0.4.8 (#396)

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac (using curl):

```bash
# download and extract the binary
<<<<<<< HEAD
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.9/argocd-autopilot-darwin-amd64.tar.gz | tar zx
=======
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.8/argocd-autopilot-darwin-amd64.tar.gz | tar zx
>>>>>>> Release-v0.4.8 (#396)

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Docker:

When using the Docker image, you have to provide the `.kube` and `.gitconfig` directories as mounts to the running container:

```
docker run \
  -v ~/.kube:/home/autopilot/.kube \
  -v ~/.gitconfig:/home/autopilot/.gitconfig \
<<<<<<< HEAD
  -it quay.io/argoprojlabs/argocd-autopilot:v0.4.9 <cmd> <flags>
=======
  -it quay.io/argoprojlabs/argocd-autopilot:v0.4.8 <cmd> <flags>
>>>>>>> Release-v0.4.8 (#396)
```

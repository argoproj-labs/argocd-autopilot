### Features
* Added a --force flag to `argocd-autopilot repo uninstall`: not allowing cluster and repo deletion errors from stopping the uninstall flow.
* Added a --fast-exit flag to `argocd-autopilot repo uninstall`: not waiting for full deletion of cluster resources.

### Contributors:
- Elad Laor ([@elad-codefresh](https://github.com/elad-codefresh))

## Installation:

To use the `argocd-autopilot` CLI you need to download the latest binary from the [git release page](https://github.com/argoproj-labs/argocd-autopilot/releases).

### Using brew:
```bash
# install
brew install argocd-autopilot

# check the installation
argocd-autopilot version
```

### Linux and WSL (using curl):
```bash
# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.2.22/argocd-autopilot-linux-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac (using curl):
```bash
# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.2.22/argocd-autopilot-darwin-amd64.tar.gz | tar zx

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
  -it quay.io/argoprojlabs/argocd-autopilot:v0.2.22 <cmd> <flags>
```

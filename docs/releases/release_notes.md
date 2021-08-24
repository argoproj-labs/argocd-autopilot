### New Features:
* Allow adding labels for the ArgoCD app created during the bootstrap [#159](https://github.com/argoproj-labs/argocd-autopilot/issues/159)
  
### Dependencies:
* Bump k8s.io/api from 0.21.1 to 0.21.3 [#135](https://github.com/argoproj-labs/argocd-autopilot/pull/135)
* Bump k8s.io/kubectl from 0.21.1 to 0.21.3 [#137](https://github.com/argoproj-labs/argocd-autopilot/pull/137)
* Bump github.com/briandowns/spinner from 1.13.0 to 1.16.0 [#149](https://github.com/argoproj-labs/argocd-autopilot/pull/149)

### Contributors:
- oren-codefresh ([@oren-codefresh](https://github.com/oren-codefresh))

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
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.2.14/argocd-autopilot-linux-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac (using curl):
```bash
# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.2.14/argocd-autopilot-darwin-amd64.tar.gz | tar zx

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
  -it quay.io/argoprojlabs/argocd-autopilot <cmd> <flags>
```

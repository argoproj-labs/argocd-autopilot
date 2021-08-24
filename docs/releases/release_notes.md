### New Features:
* Allow installation of Argo-CD in `insecure` mode (useful when you want the SSL termination to happen in the ingress controller)[#144](https://github.com/argoproj-labs/argocd-autopilot/issues/144)
  
### Breaking Changes:
* Removed the `--namespaced` option from `repo bootstrap`. Installing argo-cd in namespaced mode cannot be used for bootstraping as the bootstrap installation contains CRDs, which are cluster scoped resources, which cannot be created by argo-cd in namespaced mode. Bottom line: it was never useable.

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

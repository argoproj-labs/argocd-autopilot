### Changes
* importing argoproj/applicationset separately 
* bumping argo-cd version to v2.3.3
* removed `kube-context` from being sent to `argocd login` command

### Contributors:
- Noam Gal ([@noam-codefresh](https://github.com/noam-codefresh))
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
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.3.3/argocd-autopilot-linux-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac (using curl):
```bash
# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.3.3/argocd-autopilot-darwin-amd64.tar.gz | tar zx

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
  -it quay.io/argoprojlabs/argocd-autopilot:v0.3.3 <cmd> <flags>
```

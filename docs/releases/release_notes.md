### Bug fixes:
* Typo in error message [#162](https://github.com/argoproj-labs/argocd-autopilot/pull/162)
* The cluster-resources application-set should not have an empty project value [#165](https://github.com/argoproj-labs/argocd-autopilot/issues/165)
* Creating a DirApp with an existing name overwrites the previous application [#158](https://github.com/argoproj-labs/argocd-autopilot/issues/158)
* Creating a DirApp with no path in the repo causes app to not be created in the cluster [#166](https://github.com/argoproj-labs/argocd-autopilot/issues/166)

### Dependencies:
* Default bootstrap now installs Argo-CD 2.1.1 and ApplicationSet 0.2.0 [#168](https://github.com/argoproj-labs/argocd-autopilot/pull/168)


### Contributors:
- Laurent Rochette ([@lrochette](https://github.com/lrochette))
- Noam Gal ([@noam-codefresh](https://github.com/noam-codefresh))

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

### Breaking Changes:
* Removed `repo create` command. From now on, the `repo bootstrap` command will automatically create the repository if it currently does not exist. A new `--provider` flag was added to this command, in order to specificy the git cloud provider to use when creating the repository. Autopilot currently only supports github. Without the flag value, autopilot will try to infer the provider from the repo URL. [116](https://github.com/argoproj-labs/argocd-autopilot/pull/116)

### New Features:
* The `app create` now supports waiting for the Application to be fully Synced to the k8s cluster. The standard kubeclient flags were added in order to specificy which context is expected to recieve the new Application, and a `--timeout` flag can set the duration to wait before returning an error. The default value of 0 will not perform any wait, nor require access to the cluster at all. [117](https://github.com/argoproj-labs/argocd-autopilot/pull/117)

### Contributors:
- Roi Kramer ([@roi-codefresh](https://github.com/roi-codefresh))
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
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.2.8/argocd-autopilot-linux-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac (using curl):
```bash
# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.2.8/argocd-autopilot-darwin-amd64.tar.gz | tar zx

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

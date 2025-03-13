### Changes

- [chore] update github.com/argoproj/argo-cd/v2 v2.13.1 to v2.13.4 [#635](https://github.com/argoproj-labs/argocd-autopilot/pull/635)
- [chore] update github.com/go-git/go-billy/v5 v5.5.0 to v5.6.2 [#635](https://github.com/argoproj-labs/argocd-autopilot/pull/635)
- [chore] update github.com/go-git/go-git/v5 v5.12.0 to v5.13.2 [#635](https://github.com/argoproj-labs/argocd-autopilot/pull/635)
- [chore] replace github.com/xanzy/go-gitlab v0.91.1 with gitlab.com/gitlab-org/api/client-go v0.121.0 [#635](https://github.com/argoproj-labs/argocd-autopilot/pull/635)
- [chore] update sigs.k8s.io/kustomize/api v0.17.2 to v0.19.0 [#635](https://github.com/argoproj-labs/argocd-autopilot/pull/635)
- [chore] update sigs.k8s.io/kustomize/kyaml v0.17.1 to v0.19.0 [#635](https://github.com/argoproj-labs/argocd-autopilot/pull/635)
- [chore] update golang to 1.24 [#634](https://github.com/argoproj-labs/argocd-autopilot/pull/634)
- [feat] add cluster-only uninstall option and resource deletion handling [#634](https://github.com/argoproj-labs/argocd-autopilot/pull/634)

### Contributors:

- Noam Gal ([@ATGardner](https://github.com/ATGardner))

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
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.19/argocd-autopilot-linux-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac (using curl):

```bash
# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.19/argocd-autopilot-darwin-amd64.tar.gz | tar zx

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
  -it quay.io/argoprojlabs/argocd-autopilot:v0.4.18 <cmd> <flags>
```

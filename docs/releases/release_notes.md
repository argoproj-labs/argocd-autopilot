### Changes

- [fix] Fix ArgoCD 3.0.0+ compatibility by replacing legacy repo credentials with proper repo-creds secret [#674](https://github.com/argoproj-labs/argocd-autopilot/pull/674)
- [fix] repo bootstrap fails when repository does not exist in the git provider [#675](https://github.com/argoproj-labs/argocd-autopilot/issues/675)
- [chore] upgraded github.com/argoproj/argo-cd/v2 v2.13.4 => github.com/argoproj/argo-cd/v3 v3.1.5 [#678](https://github.com/argoproj-labs/argocd-autopilot/pull/678)
- [chore] upgraded github.com/spf13/pflag v1.0.5 => v1.0.10 [#678](https://github.com/argoproj-labs/argocd-autopilot/pull/678)
- [chore] upgraded github.com/briandowns/spinner v1.23.1 => v1.23.2 [#678](https://github.com/argoproj-labs/argocd-autopilot/pull/678)
- [chore] upgraded github.com/go-jose/go-jose/v4 v4.0.4 => v4.1.2 [#678](https://github.com/argoproj-labs/argocd-autopilot/pull/678)
- [chore] upgraded github.com/redis/go-redis/v9 v9.6.1 => v9.14.0 [#678](https://github.com/argoproj-labs/argocd-autopilot/pull/678)
- [chore] upgraded code.gitea.io/sdk/gitea v0.19.0 => v0.22.0 [#678](https://github.com/argoproj-labs/argocd-autopilot/pull/678)
- [chore] upgraded gitlab.com/gitlab-org/api/client-go v0.121.0 => v0.143.3 [#678](https://github.com/argoproj-labs/argocd-autopilot/pull/678)
- [chore] update to golang 1.25.1 [#678](https://github.com/argoproj-labs/argocd-autopilot/pull/678)
- [chore] updated golangci-lint to 2.4.0 [#678](https://github.com/argoproj-labs/argocd-autopilot/pull/678)

### Contributors:

- Aron Reis ([@aronreisx](https://github.com/aronreisx))
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
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.20/argocd-autopilot-linux-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac (using curl):

```bash
# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.20/argocd-autopilot-darwin-amd64.tar.gz | tar zx

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

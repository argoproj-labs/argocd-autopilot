### Changes

- [chore] Bump code.gitea.io/sdk/gitea from 0.15.1 to 0.16.0 [#503](https://github.com/argoproj-labs/argocd-autopilot/pull/503)
- [chore] Bump github.com/argoproj/argo-cd/v2 from 2.8.3 to 2.8.4 [#503](https://github.com/argoproj-labs/argocd-autopilot/pull/503)
- [chore] Bump github.com/briandowns/spinner from 1.18.1 to 1.23.0 [#499](https://github.com/argoproj-labs/argocd-autopilot/pull/499)
- [chore] Bump github.com/go-git/go-git/v5 from 5.7.0 to 5.9.0 [#505](https://github.com/argoproj-labs/argocd-autopilot/pull/505)
- [chore] Bump github.com/ktrysmt/go-bitbucket from 0.9.60 to 0.9.70 [#502](https://github.com/argoproj-labs/argocd-autopilot/pull/502) [#506](https://github.com/argoproj-labs/argocd-autopilot/pull/506) [#514](https://github.com/argoproj-labs/argocd-autopilot/pull/514)
- [chore] Bump github.com/spf13/viper from 1.16.0 to 1.17.0 [#511](https://github.com/argoproj-labs/argocd-autopilot/pull/511)
- [chore] Bump github.com/xanzy/go-gitlab from 0.86.0 to 0.93.1 [#500](https://github.com/argoproj-labs/argocd-autopilot/pull/500) [#508](https://github.com/argoproj-labs/argocd-autopilot/pull/508) [#512](https://github.com/argoproj-labs/argocd-autopilot/pull/512)
- [chore] Bump golang.org/x/net from 0.15.0 to 0.17.0 [#513](https://github.com/argoproj-labs/argocd-autopilot/pull/513)
- [chore] Bump k8s.io/* deps to v0.24.17 [#513](https://github.com/argoproj-labs/argocd-autopilot/pull/513)
- [chore] removed git-lfs from docker image [#513](https://github.com/argoproj-labs/argocd-autopilot/pull/513)

### Contributors:

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
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.18/argocd-autopilot-linux-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac (using curl):

```bash
# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.18/argocd-autopilot-darwin-amd64.tar.gz | tar zx

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

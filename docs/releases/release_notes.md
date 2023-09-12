### Changes

- [fix] Fix gitea repo bootstrap failure [#495](https://github.com/argoproj-labs/argocd-autopilot/pull/495)
- [chore] updated image to `1.20.3` [#477](https://github.com/argoproj-labs/argocd-autopilot/pull/477)
- [chore] updated golangci-lint from `v1.50.1` to `v1.53.1` [#479](https://github.com/argoproj-labs/argocd-autopilot/pull/479)
- [chore] Bump github.com/spf13/viper from `1.10.1` to `1.16.0` [#475](https://github.com/argoproj-labs/argocd-autopilot/pull/475)
- [chore] Bump go.mongodb.org/mongo-driver from `1.1.2` to `1.5.1` [#433](https://github.com/argoproj-labs/argocd-autopilot/pull/433)
- [chore] Bump github.com/xanzy/go-gitlab from `0.71.0` to `0.86.0` [#487](https://github.com/argoproj-labs/argocd-autopilot/pull/487)
- [chore] Bump k8s.io/* from `0.24.2` to `0.24.15` [#489](https://github.com/argoproj-labs/argocd-autopilot/pull/489)
- [chore] Bump github.com/argoproj/argo-cd/v2 from `2.5.9` to `2.8.3` [#488](https://github.com/argoproj-labs/argocd-autopilot/pull/488)
- [chore] Bump github.com/go-git/go-billy/v5 `5.3.1` to `5.4.1` [#488](https://github.com/argoproj-labs/argocd-autopilot/pull/488)
- [chore] Bump github.com/go-git/go-git/v5 `5.4.2` to `5.7.0` [#488](https://github.com/argoproj-labs/argocd-autopilot/pull/488)
- [chore] Bump github.com/ktrysmt/go-bitbucket `0.9.55` to `0.9.60` [#488](https://github.com/argoproj-labs/argocd-autopilot/pull/488)
- [chore] Bump github.com/sirupsen/logrus `1.8.1` to `1.9.3` [#488](https://github.com/argoproj-labs/argocd-autopilot/pull/488)
- [chore] Bump github.com/spf13/cobra `1.5.0` to `1.7.0` [#488](https://github.com/argoproj-labs/argocd-autopilot/pull/488)
- [chore] Bump sigs.k8s.io/kustomize/api `0.11.4` to `0.11.5` [#488](https://github.com/argoproj-labs/argocd-autopilot/pull/488)
- [chore] Bump sigs.k8s.io/kustomize/kyaml `0.13.6` to `0.13.7` [#488](https://github.com/argoproj-labs/argocd-autopilot/pull/488)
- [chore] Bump github.com/go-git/go-billy/v5 from `5.4.1` to `5.5.0` [#497](https://github.com/argoproj-labs/argocd-autopilot/pull/497)
- [chore] Bump pygments from `2.7.4` to `2.15.0` in /docs [#493](https://github.com/argoproj-labs/argocd-autopilot/pull/493)
- [docs] add missing requirement in getting started docs [#459](https://github.com/argoproj-labs/argocd-autopilot/pull/459)
- [docs] fix a typo [#460](https://github.com/argoproj-labs/argocd-autopilot/pull/460)
- [docs] Quoting variable expansions [#391](https://github.com/argoproj-labs/argocd-autopilot/pull/391)

### Contributors:

- Wim Fournier ([@hsmade](https://github.com/hsmade))
- Noam Gal ([@noam-codefresh](https://github.com/noam-codefresh))
- gamerslouis ([@gamerslouis](https://github.com/gamerslouis))
- TomyLobo ([@TomyLobo](https://github.com/TomyLobo))

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
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.15/argocd-autopilot-linux-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac (using curl):

```bash
# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/v0.4.15/argocd-autopilot-darwin-amd64.tar.gz | tar zx

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
  -it quay.io/argoprojlabs/argocd-autopilot:v0.4.15 <cmd> <flags>
```

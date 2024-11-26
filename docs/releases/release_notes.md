### Changes

- [chore] update golang to 1.23 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] code.gitea.io/sdk/gitea v0.17.1 => v0.19.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded github.com/argoproj/argo-cd/v2 v2.10.0 => v2.13.1 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded github.com/briandowns/spinner v1.23.0 => v1.23.1 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded github.com/go-git/go-git/v5 v5.11.0 => v5.12.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded github.com/ktrysmt/go-bitbucket v0.9.75 => v0.9.81 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded github.com/spf13/cobra v1.8.0 => v1.8.1 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded github.com/spf13/viper v1.18.2 => v1.19.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded github.com/stretchr/testify v1.8.4 => v1.10.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded github.com/xanzy/go-gitlab v0.97.0 => v0.114.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded k8s.io/api v0.26.11 => v0.31.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded k8s.io/apimachinery v0.26.11 => v0.31.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded k8s.io/cli-runtime v0.26.11 => v0.31.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded k8s.io/client-go v0.26.11 => v0.31.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded k8s.io/kubectl v0.26.11 => v0.31.2 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded sigs.k8s.io/kustomize/api v0.12.1 => v0.17.2 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] upgraded sigs.k8s.io/kustomize/kyaml v0.13.9 => v0.17.1 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] replaced github.com/ghodss/yaml v1.0.0 with sigs.k8s.io/yaml v1.4.0 [#598](https://github.com/argoproj-labs/argocd-autopilot/pull/598)
- [chore] Bump go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc [#525](https://github.com/argoproj-labs/argocd-autopilot/pull/525)
- [chore] Bump github.com/go-jose/go-jose/v3 from 3.0.0 to 3.0.1 [#528](https://github.com/argoproj-labs/argocd-autopilot/pull/528)
- [chore] Bump golang.org/x/crypto from 0.16.0 to 0.17.1 [#542](https://github.com/argoproj-labs/argocd-autopilot/pull/542) [#544](https://github.com/argoproj-labs/argocd-autopilot/pull/544)
- [chore] Bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 [#546](https://github.com/argoproj-labs/argocd-autopilot/pull/546)

### Contributors:

- Noam Gal ([@noam-codefresh](https://github.com/noam-codefresh))
- [priyanshusd](https://github.com/priyanshusd)

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

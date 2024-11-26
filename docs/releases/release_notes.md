### Changes

- [chore] Bump github.com/spf13/cobra from 1.7.0 to 1.8.0 [#523](https://github.com/argoproj-labs/argocd-autopilot/pull/523)
- [chore] Bump go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc [#525](https://github.com/argoproj-labs/argocd-autopilot/pull/525)
- [chore] Bump github.com/go-jose/go-jose/v3 from 3.0.0 to 3.0.1 [#528](https://github.com/argoproj-labs/argocd-autopilot/pull/528)
- [chore] Bump code.gitea.io/sdk/gitea from 0.16.0 to 0.17.1 [#532](https://github.com/argoproj-labs/argocd-autopilot/pull/532) [#544](https://github.com/argoproj-labs/argocd-autopilot/pull/544)
- [chore] Bump github.com/go-git/go-git/v5 from 5.9.0 to 5.11.0 [#519](https://github.com/argoproj-labs/argocd-autopilot/pull/519) [#531](https://github.com/argoproj-labs/argocd-autopilot/pull/531) [#539](https://github.com/argoproj-labs/argocd-autopilot/pull/539)
- [chore] Bump github.com/spf13/viper from 1.17.0 to 1.18.2 [#537](https://github.com/argoproj-labs/argocd-autopilot/pull/537) [#543](https://github.com/argoproj-labs/argocd-autopilot/pull/543)
- [chore] Bump golang.org/x/crypto from 0.16.0 to 0.17.1 [#542](https://github.com/argoproj-labs/argocd-autopilot/pull/542) [#544](https://github.com/argoproj-labs/argocd-autopilot/pull/544)
- [chore] Bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 [#546](https://github.com/argoproj-labs/argocd-autopilot/pull/546)
- [chore] Bump github.com/argoproj/argo-cd/v2 from 2.8.4 to 2.10.0 [#554](https://github.com/argoproj-labs/argocd-autopilot/pull/554)
- [chore] Bump k8s.io/* from 0.24.17 to 0.26.11 [#554](https://github.com/argoproj-labs/argocd-autopilot/pull/554)
- [chore] Bump sigs.k8s.io/kustomize/api from 0.11.5 to 0.12.1[#554](https://github.com/argoproj-labs/argocd-autopilot/pull/554)
- [chore] Bump sigs.k8s.io/kustomize/kyaml from 0.13.7 to 0.13.9 [#554](https://github.com/argoproj-labs/argocd-autopilot/pull/554)
- [chore] Bump github.com/xanzy/go-gitlab from 0.93.1 to 0.97.0 [#518](https://github.com/argoproj-labs/argocd-autopilot/pull/518) [#526](https://github.com/argoproj-labs/argocd-autopilot/pull/526) [#541](https://github.com/argoproj-labs/argocd-autopilot/pull/541) [#552](https://github.com/argoproj-labs/argocd-autopilot/pull/552)
- [chore] Bump github.com/ktrysmt/go-bitbucket from 0.9.70 to 0.9.75 [#524](https://github.com/argoproj-labs/argocd-autopilot/pull/524) [#527](https://github.com/argoproj-labs/argocd-autopilot/pull/527) [#540](https://github.com/argoproj-labs/argocd-autopilot/pull/540) [#553](https://github.com/argoproj-labs/argocd-autopilot/pull/553)
- [chore] updated golang to 1.22 [#555](https://github.com/argoproj-labs/argocd-autopilot/pull/555)

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

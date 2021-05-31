# Installation of the CLI

To use the `argocd-autopilot` CLI you need to install the latest binary with [Homebrew](https://brew.sh/) (package manager) or to download the latest binary from the [git release page](https://github.com/argoproj-labs/argocd-autopilot/releases).

## Linux and WSL
### Install with Homebrew
Install the latest binary:
```bash
brew install argocd-autopilot
```

Check the installation:
```bash
argocd-autopilot version
```

### Download with Curl
You can view the latest version of Argo CD Autopilot at the link above or run the following command to grab the version:
```bash
VERSION=$(curl --silent "https://api.github.com/repos/argoproj-labs/argocd-autopilot/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
```

Replace `VERSION` in the command below with the version of Argo CD Autopilot you would like to download:
```bash
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/$VERSION/argocd-autopilot-linux-amd64.tar.gz | tar zx
```

Move the `argocd-autopilot` binary to your $PATH:
```bash
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot
```

Check the installation:
```bash
argocd-autopilot version
```

## Mac
### Install with Homebrew
Install the latest binary:
```bash
brew install argocd-autopilot
```

Check the installation:
```bash
argocd-autopilot version
```

### Download With Curl
You can view the latest version of Argo CD Autopilot at the link above or run the following command to grab the version:
```bash
VERSION=$(curl --silent "https://api.github.com/repos/argoproj-labs/argocd-autopilot/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
```

Replace `VERSION` in the command below with the version of Argo CD Autopilot you would like to download:
```bash
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/$VERSION/argocd-autopilot-darwin-amd64.tar.gz | tar zx
```

Move the `argocd-autopilot` binary to your $PATH:
```bash
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot
```

Check the installation:
```bash
argocd-autopilot version
```

## Docker
When using the Docker image, you have to provide the `.kube` and `.gitconfig` directories as mounts to the running container:
```
docker run \
  -v ~/.kube:/home/autopilot/.kube \
  -v ~/.gitconfig:/home/autopilot/.gitconfig \
  -it quay.io/argoprojlabs/argocd-autopilot <cmd> <flags>
```

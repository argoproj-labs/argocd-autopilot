### Bug fixes:

* `--project` flag shows in unrelated commands and not marked as required where it should be.

### Additional changes

* Added `brew` formula for `argocd-autopilot` [#31](https://github.com/argoproj-labs/argocd-autopilot/issues/31)

### Contributors:

- Stéphane Este-Gracias ([@sestegra](https://github.com/sestegra))
- Noam Gal ([@noam-codefresh](https://github.com/noam-codefresh))
- Roi Kramer ([@roi-codefresh](https://github.com/roi-codefresh))

## Installation:

To use the `argocd-autopilot` CLI you need to download the latest binary from the [git release page](https://github.com/argoproj-labs/argocd-autopilot/releases).

### Linux
```bash
# get the latest version or change to a specific version
VERSION=$(curl --silent "https://api.github.com/repos/argoproj-labs/argocd-autopilot/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/$VERSION/argocd-autopilot-linux-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

### Mac
#### Using brew
```bash
# install
brew install argocd-autopilot

# check the installation
argocd-autopilot version
```

#### Using curl
```bash
# get the latest version or change to a specific version
VERSION=$(curl --silent "https://api.github.com/repos/argoproj-labs/argocd-autopilot/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

# download and extract the binary
curl -L --output - https://github.com/argoproj-labs/argocd-autopilot/releases/download/$VERSION/argocd-autopilot-darwin-amd64.tar.gz | tar zx

# move the binary to your $PATH
mv ./argocd-autopilot-* /usr/local/bin/argocd-autopilot

# check the installation
argocd-autopilot version
```

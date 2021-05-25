### Bug fixes:

* Fixed `--namespaced` bootstrap [#61](https://github.com/argoproj-labs/argocd-autopilot/pull/61)
* fix typo in auth error message [#60](https://github.com/argoproj-labs/argocd-autopilot/pull/60)

### Features:

* Support for directory type application [#59](https://github.com/argoproj-labs/argocd-autopilot/issues/59)

### Additional changes

* Renamed the binary archive from just .gz zo .tar.gz [#62](https://github.com/argoproj-labs/argocd-autopilot/pull/62)

### Contributors:

- Engin Diri ([@dirien](https://github.com/dirien))
- Christopher Baklid ([@inveracity](https://github.com/inveracity))

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

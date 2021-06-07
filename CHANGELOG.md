# Changelog:

# v0.2.0

### Breaking Changes:
* Combined `--repo`, `--installation-path` and `--revision` into a single url, set by `--repo` with the following syntax:  
```
argocd-autopilot <command> --repo https://github.com/owner/name/path/to/installation_path?ref=branch
```
The `REPO_URL` environment variable also uses the new syntax

### Bug fixes:
* failed to build bootstrap manifests [#82](https://github.com/argoproj-labs/argocd-autopilot/issues/82)
* Adding two applications with the same ns causes sync ping-pong [#23](https://github.com/argoproj-labs/argocd-autopilot/issues/23)

### Additional changes:
* The `RunRepoCreate` func now returns `(*git.CloneOptions, error)`

# v0.1.10

### Bug fixes:

* removed dependency on `packr` for compiling source with additional assets required by argo-cd dependency.

# v0.1.9
### Bug fixes:

* `--project` flag shows in unrelated commands and not marked as required where it should be.

### Additional changes

* Added `brew` formula for `argocd-autopilot` [#31](https://github.com/argoproj-labs/argocd-autopilot/issues/31)

# v0.1.8

* Fix -p option in README.md
* renamed module from `argoproj/argocd-autopilot` to `argoproj-labs/argocd-autopilot`
# v0.1.7

* Fixed `--namespaced` bootstrap
* fix typo in auth error message
* Support for directory type application
* Renamed the binary archive from just .gz zo .tar.gz

# v0.1.6
* new logo!
* updated docs
# v0.1.5
* doc fixes
* you no longer need to run `argocd login` after running `repo bootstrap`
* added `app delete` command

# v0.1.4
* doc fixes
* fixed adding application on a remote cluster

# v0.1.3
* fixed docker image creation

# v0.1.2
* added documentation
* improved CI-CD flow

# v0.1.0
This is the first release of the `argocd-autopilot` tool.

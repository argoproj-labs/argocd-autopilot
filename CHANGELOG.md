# Changelog:

# v0.2.6

### Bug Fixes:
* getting "failed to build bootstrap manifests" since v0.2.5 [#106](https://github.com/argoproj-labs/argocd-autopilot/issues/106)

### Breaking Changes:
* ~when sending `--app` flag value, use either `?sha=<sha_value>`, `?tag=<tag_name>` or `?ref=<branch_name>` to specificy sha|tag|branch to clone from ~ [#98](https://github.com/argoproj-labs/argocd-autopilot/pull/98)~ - REVERTED in [#107](https://github.com/argoproj-labs/argocd-autopilot/pull/107)

### Additional Changes:
* fixed help text typos (#105)[https://github.com/argoproj-labs/argocd-autopilot/pull/105]

# v0.2.2

### Bug fixes:
* App type infer fails when --app value references a tag [#97](https://github.com/argoproj-labs/argocd-autopilot/issues/97)
* Deleting the bootstrap app hangs while deleting the entire hierarchy [#99](https://github.com/argoproj-labs/argocd-autopilot/issues/99)

### Breaking Changes:
* when sending `--app` flag value, use either `?sha=<sha_value>`, `?tag=<tag_name>` or `?ref=<branch_name>` to specificy sha|tag|branch to clone from [#98](https://github.com/argoproj-labs/argocd-autopilot/pull/98)

### Additional changes:
* update docs about secrets not yet supported [#93](https://github.com/argoproj-labs/argocd-autopilot/pull/93)
* Support using 2 repos for Kustomize apps [#97](https://github.com/argoproj-labs/argocd-autopilot/issues/97)

# v0.2.1

### Bug fixes:
* app create does not work with local path (tries to infer application type by cloning) [#87](https://github.com/argoproj-labs/argocd-autopilot/issues/87)
* Clone logs not displaying correct values
* Debug logs not showing

### Additional changes:
* Updated k8s dependencies from v0.20.4 to v0.21.1
* Added `--progress` flag to redirect the git operations
* `CloneOptions.FS` is now `fs.FS` instead of `billy.Filesystem`

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

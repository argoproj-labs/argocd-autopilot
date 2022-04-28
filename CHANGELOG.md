# Changelog:

# v0.3.6
* fixed extracting email from github token
* support labels for bootstrap namespace

# v0.3.5
### Changes
* adding flags for include/exclude files for app create

# v0.3.4
### bumping version to fix release (no code changes)

# v0.3.3
### Changes
* importing argoproj/applicationset separately 
* bumping argo-cd version to v2.3.3
* removed `kube-context` from being sent to `argocd login` command

# v0.3.2

### Changes
* Updated `project create` to include `--labels` flag
* Updated `app create` to include `--labels` flag
* format `config.json` and `config_dir.json` for readability
* extract committer username and email from token, as a fallback from git config data
* fixed azure repo bootstrap failure [#267](https://github.com/argoproj-labs/argocd-autopilot/issues/267)
* send `--kube-context` to `argocd login` command

# v0.3.1

### Changes
* Added `SECURITY.md` file. [#253](https://github.com/argoproj-labs/argocd-autopilot/issues/253)
* Updated `golangci-lint` to `v1.45.2`. [#256](https://github.com/argoproj-labs/argocd-autopilot/pull/256)
* Moved from `mockery` tool to `mockgen`. [#256](https://github.com/argoproj-labs/argocd-autopilot/pull/256)
* Added `snyk` security scan in CI and Release pipelines. [#256](https://github.com/argoproj-labs/argocd-autopilot/pull/256)
* Bumped `github.com/argoproj/argo-cd` dependency to `v2.3.2`. [#256](https://github.com/argoproj-labs/argocd-autopilot/pull/256)


# v0.3.0

### Security Fixes
* Moved to using Argo-CD and ApplicationSet stable versions (currently: argocd-v2.3.1 and applicationset-v0.4.1). [#247](https://github.com/argoproj-labs/argocd-autopilot/pull/247)

### Changes
* Removed dependency on `argoproj-labs/applicationset` and moved to install it from https://github.com/argoproj/argo-cd/tree/master/manifests/addons/applicationset. [#247](https://github.com/argoproj-labs/argocd-autopilot/pull/247)

# v0.2.31

### Features
* Added `--force` flag to the `repo uninstall` command which allows you to skip failed steps in the uninstall process to ensure a clean cluster. [#243](https://github.com/argoproj-labs/argocd-autopilot/pull/243)

### Bug Fixes
* Fixed "already up-to-date" error when trying to bootstrap on a different branch. [#243](https://github.com/argoproj-labs/argocd-autopilot/pull/243)

# v0.2.30

### Features
* Added bootstrap apps labels. [#235](https://github.com/argoproj-labs/argocd-autopilot/pull/235)

# v0.2.29

### Bug Fixes
* Ignore `resource not found` errors when running `argocd-autopilot repo uninstall`. [#234](https://github.com/argoproj-labs/argocd-autopilot/pull/234)

# v0.2.28

### Features
* Added support for Azure DevOps git provider. [#217](https://github.com/argoproj-labs/argocd-autopilot/pull/217)

# v0.2.27

### Features
* Added option to create a git-src repo in case it doesn't already exist. [#209](https://github.com/argoproj-labs/argocd-autopilot/pull/212)

# v0.2.26

### Features
* Added support for gitlab git provider. [#209](https://github.com/argoproj-labs/argocd-autopilot/pull/209)

# v0.2.25

### Bug Fixes
* Fixed failed installation into a repo sub-directory [#206](https://github.com/argoproj-labs/argocd-autopilot/pull/206)

# v0.2.24

### Features
* Added `--context` flag to `repo bootstrap`, `repo uninstall` and `app create`. [#199](https://github.com/argoproj-labs/argocd-autopilot/pull/199)

# v0.2.23

### Features
* Use default branch from git config instead of master [#196](https://github.com/argoproj-labs/argocd-autopilot/pull/196)

# v0.2.22

### Features
* Added a --force flag to `argocd-autopilot repo uninstall`: not allowing cluster and repo deletion errors from stopping the uninstall flow.
* Added a --fast-exit flag to `argocd-autopilot repo uninstall`: not waiting for full deletion of cluster resources.

# v0.2.21

### Bug Fixes
* Fixed sporadic "repository not found" errors when cloning or pushing [#191](https://github.com/argoproj-labs/argocd-autopilot/pull/191).

# v0.2.19

### Features
* Change directory app to `recurse` mode by default [#185](https://github.com/argoproj-labs/argocd-autopilot/pull/185).

# v0.2.18

### Internal features:
* Allow controlling the output of git clone and push progress [#178](https://github.com/argoproj-labs/argocd-autopilot/pull/178)

# v0.2.17

### Bug fixes:
* Getting Cluster.server has invalid type: got string, want pointer when creating project on a different cluster [#154](https://github.com/argoproj-labs/argocd-autopilot/issues/154)

# v0.2.16

### Bug fixes:
* Support Bitbucket stash as provider. [#171](https://github.com/argoproj-labs/argocd-autopilot/issues/171)
  Added `--git-user` flag (and `GIT_USER` env variable) to support sending username for git authentication during clone/push

# v0.2.15

### Bug fixes:
* Typo in error message [#162](https://github.com/argoproj-labs/argocd-autopilot/pull/162)
* The cluster-resources application-set should not have an empty project value [#165](https://github.com/argoproj-labs/argocd-autopilot/issues/165)
* Creating a DirApp with an existing name overwrites the previous application [#158](https://github.com/argoproj-labs/argocd-autopilot/issues/158)
* Creating a DirApp with no path in the repo causes app to not be created in the cluster [#166](https://github.com/argoproj-labs/argocd-autopilot/issues/166)

### Dependencies:
* Default bootstrap now installs Argo-CD 2.1.1 and ApplicationSet 0.2.0 [#168](https://github.com/argoproj-labs/argocd-autopilot/pull/168)

# v0.2.14

### New featues:
* Allow adding labels for the ArgoCD app created during the bootstrap [#159](https://github.com/argoproj-labs/argocd-autopilot/issues/159)

### Dependencies:
* Bump k8s.io/api from 0.21.1 to 0.21.3 [#135](https://github.com/argoproj-labs/argocd-autopilot/pull/135)
* Bump k8s.io/kubectl from 0.21.1 to 0.21.3 [#137](https://github.com/argoproj-labs/argocd-autopilot/pull/137)
* Bump github.com/briandowns/spinner from 1.13.0 to 1.16.0 [#149](https://github.com/argoproj-labs/argocd-autopilot/pull/149)

# v0.2.13

### New Features:
* Allow installation of Argo-CD in `insecure` mode (useful when you want the SSL termination to happen in the ingress controller)[#144](https://github.com/argoproj-labs/argocd-autopilot/issues/144)
  
### Breaking Changes:
* Removed the `--namespaced` option from `repo bootstrap`. Installing argo-cd in namespaced mode cannot be used for bootstraping as the bootstrap installation contains CRDs, which are cluster scoped resources, which cannot be created by argo-cd in namespaced mode. Bottom line: it was never useable.

# v0.2.12

### New Features:
* Allow sending extra key-value pairs to app create [138](https://github.com/argoproj-labs/argocd-autopilot/issues/138)

### Documentation fixes:
* update url path to core_concepts docs [#141](https://github.com/argoproj-labs/argocd-autopilot/pull/141)

# v0.2.11

### Bug fixes:
* fixed provider sort order in field description text [#131](https://github.com/argoproj-labs/argocd-autopilot/pull/131)

# v0.2.10

### New Features:
* Support gitea as a SCM provider [#129](https://github.com/argoproj-labs/argocd-autopilot/issues/129)

### Bug fixes:
* `repo bootstrap` fails when running argocd login if passing different --kubeconfig argument [#125](https://github.com/argoproj-labs/argocd-autopilot/issues/125)

# v0.2.9

### New Features:
* Add an repo uninstall command to argocd-autopilot [#42](https://github.com/argoproj-labs/argocd-autopilot/issues/42) - Running this command will delete all manifest files and directory structure from the GitOps Repository, and all the resources from the k8s cluster

### Additional Features:
* improve sanity check [#119](https://github.com/argoproj-labs/argocd-autopilot/pull/119) - runs `repo bootstrap` on every push. Also fixed ClusterRoleBinding now being applied correctly

# v0.2.8

### Breaking Changes:
* Removed `repo create` command. From now on, the `repo bootstrap` command will automatically create the repository if it currently does not exist. A new `--provider` flag was added to this command, in order to specificy the git cloud provider to use when creating the repository. Autopilot currently only supports github. Without the flag value, autopilot will try to infer the provider from the repo URL. [116](https://github.com/argoproj-labs/argocd-autopilot/pull/116)

### New Features:
* The `app create` now supports waiting for the Application to be fully Synced to the k8s cluster. The standard kubeclient flags were added in order to specificy which context is expected to recieve the new Application, and a `--wait-timeout` flag can set the duration to wait before returning an error. The default value of 0 will not perform any wait, nor require access to the cluster at all. [117](https://github.com/argoproj-labs/argocd-autopilot/pull/117)

# v0.2.7

### Bug Fixes:
* url_parse_fix [#106](https://github.com/argoproj-labs/argocd-autopilot/issues/106)

### Additional Changes:
* Fix typo [#109](https://github.com/argoproj-labs/argocd-autopilot/pull/109)

# v0.2.6

### Bug Fixes:
* getting "failed to build bootstrap manifests" since v0.2.5 [#106](https://github.com/argoproj-labs/argocd-autopilot/issues/106)

### Breaking Changes:
* ~when sending `--app` flag value, use either `?sha=<sha_value>`, `?tag=<tag_name>` or `?ref=<branch_name>` to specificy sha|tag|branch to clone from ~ [#98](https://github.com/argoproj-labs/argocd-autopilot/pull/98)~ - REVERTED in [#107](https://github.com/argoproj-labs/argocd-autopilot/pull/107)

### Additional Changes:
* fixed help text typos [#105](https://github.com/argoproj-labs/argocd-autopilot/pull/105)

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

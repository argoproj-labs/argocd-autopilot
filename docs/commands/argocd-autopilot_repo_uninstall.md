## argocd-autopilot repo uninstall

Uninstalls an installation

```
argocd-autopilot repo uninstall [flags]
```

### Examples

```

# To run this command you need to create a personal access token for your git provider
# and provide it using:

    export GIT_TOKEN=<token>

# or with the flag:

    --git-token <token>

# Uninstall argo-cd from the current kubernetes context in the argocd namespace
# and delete all manifests rom the root of gitops repository

    argocd-autopilot repo uninstall --repo https://github.com/example/repo

# Uninstall argo-cd from the current kubernetes context in the argocd namespace
# and delete all manifests from a specific folder in the gitops repository

    argocd-autopilot repo uninstall --repo https://github.com/example/repo/path/to/installation_root

```

### Options

```
  -t, --git-token string         Your git provider api token [GIT_TOKEN]
  -h, --help                     help for uninstall
      --kubeconfig string        Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string         If present, the namespace scope for this CLI request
      --repo string              Repository URL [GIT_REPO]
      --request-timeout string   The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
```

### SEE ALSO

* [argocd-autopilot repo](argocd-autopilot_repo.md)	 - Manage gitops repositories


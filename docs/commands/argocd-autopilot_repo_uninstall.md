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

# Uninstall using the --force flag will try to uninstall even if some steps
# failed. For example, if it cannot clone the bootstrap repo for some reason
# it will still attempt to delete argo-cd from the cluster. Use with caution!

    argocd-autopilot repo uninstall --repo https://github.com/example/repo --force

```

### Options

```
      --context string           The name of the kubeconfig context to use
      --force                    If true, will try to complete the uninstallation even if one or more of the uninstallation steps failed
      --git-server-crt string    Git Server certificate file
  -t, --git-token string         Your git provider api token [GIT_TOKEN]
  -u, --git-user string          Your git provider user name [GIT_USER] (not required in GitHub)
  -h, --help                     help for uninstall
      --kubeconfig string        Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string         If present, the namespace scope for this CLI request
      --repo string              Repository URL [GIT_REPO]
      --request-timeout string   The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -b, --upsert-branch            If true will try to checkout the specified branch and create it if it doesn't exist
```

### SEE ALSO

* [argocd-autopilot repo](argocd-autopilot_repo.md)	 - Manage gitops repositories


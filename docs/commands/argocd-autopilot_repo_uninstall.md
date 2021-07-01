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
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "/home/user/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
  -t, --git-token string               Your git provider api token [GIT_TOKEN]
  -h, --help                           help for uninstall
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
      --repo string                    Repository URL [GIT_REPO]
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

### SEE ALSO

* [argocd-autopilot repo](argocd-autopilot_repo.md)	 - Manage gitops repositories


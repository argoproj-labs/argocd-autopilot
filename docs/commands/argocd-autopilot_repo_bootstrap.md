## argocd-autopilot repo bootstrap

Bootstrap a new installation

```
argocd-autopilot repo bootstrap [flags]
```

### Examples

```

# To run this command you need to create a personal access token for your git provider
# and provide it using:
    
    export GIT_TOKEN=<token>

# or with the flag:
    
    --token <token>
        
# Install argo-cd on the current kubernetes context in the argocd namespace
# and persists the bootstrap manifests to the root of gitops repository
    
    argocd-autopilot repo bootstrap --repo https://github.com/example/repo

    # Install argo-cd on the current kubernetes context in the argocd namespace
    # and persists the bootstrap manifests to a specific folder in the gitops repository

    argocd-autopilot repo bootstrap --repo https://github.com/example/repo --installation-path path/to/bootstrap/root

```

### Options

```
      --app string                     The application specifier (e.g. argocd@v1.0.2)
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "/Users/roikramer/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --dry-run                        If true, print manifests instead of applying them to the cluster (nothing will be commited to git)
  -t, --git-token string               Your git provider api token [GIT_TOKEN]
  -h, --help                           help for bootstrap
      --hide-password                  If true, will not print initial argo cd password
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --installation-mode string       One of: normal|flat. If flat, will commit the bootstrap manifests, otherwise will commit the bootstrap kustomization.yaml (default "normal")
      --installation-path string       The path where we of the installation files (defaults to the root of the repository [GIT_INSTALLATION_PATH]
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
      --namespaced                     If true, install a namespaced version of argo-cd (no need for cluster-role)
      --repo string                    Repository URL [GIT_REPO]
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
      --revision string                Repository branch, tag or commit hash (defaults to HEAD)
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

### SEE ALSO

* [argocd-autopilot repo](argocd-autopilot_repo.md)	 - Manage gitops repositories


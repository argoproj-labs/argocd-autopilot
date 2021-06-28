## argocd-autopilot application create

Create an application in a specific project

```
argocd-autopilot application create [APP_NAME] [flags]
```

### Examples

```

# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:

        export GIT_TOKEN=<token>
        export GIT_REPO=<repo_url>

# or with the flags:

        --git-token <token> --repo <repo_url>

# using the --type flag (kustomize|dir) is optional. If it is ommitted, argocd-autopilot will clone
# the --app repository, and infer the type automatically.

# Create a new application from kustomization in a remote repository (will reference the HEAD revision)

    argocd-autopilot app create <new_app_name> --app github.com/some_org/some_repo/manifests --project project_name

# Reference a specific git commit hash:

  argocd-autopilot app create <new_app_name> --app github.com/some_org/some_repo/manifests?sha=<commit_hash> --project project_name

# Reference a specific git tag:

  argocd-autopilot app create <new_app_name> --app github.com/some_org/some_repo/manifests?tag=<tag_name> --project project_name

# Reference a specific git branch:

  argocd-autopilot app create <new_app_name> --app github.com/some_org/some_repo/manifests?ref=<branch_name> --project project_name

# Wait until the application is Synced in the cluster:

  argocd-autopilot app create <new_app_name> --app github.com/some_org/some_repo/manifests --project project_name --wait-timeout 2m --context my_context 

```

### Options

```
      --app string                     The application specifier (e.g. github.com/argoproj/argo-workflows/manifests/cluster-install/?ref=v3.0.3)
      --apps-git-token string          Your git provider api token [APPS_GIT_TOKEN]
      --apps-repo string               Repository URL [APPS_GIT_REPO]
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "/home/user/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --dest-namespace string          K8s target namespace (overrides the namespace specified in the kustomization.yaml)
      --dest-server string             K8s cluster URL (e.g. https://kubernetes.default.svc) (default "https://kubernetes.default.svc")
  -h, --help                           help for create
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --installation-mode string       One of: normal|flat. If flat, will commit the application manifests (after running kustomize build), otherwise will commit the kustomization.yaml (default "normal")
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
  -p, --project string                 Project name
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --type string                    The application type (kustomize|dir)
      --user string                    The name of the kubeconfig user to use
      --wait-timeout duration          If not '0s', will try to connect to the cluster and wait until the application is in 'Synced' status for the specified timeout period
```

### Options inherited from parent commands

```
  -t, --git-token string   Your git provider api token [GIT_TOKEN]
      --repo string        Repository URL [GIT_REPO]
```

### SEE ALSO

* [argocd-autopilot application](argocd-autopilot_application.md)	 - Manage applications


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
      --app string                 The application specifier (e.g. github.com/argoproj/argo-workflows/manifests/cluster-install/?ref=v3.0.3)
      --apps-git-token string      Your git provider api token [APPS_GIT_TOKEN]
      --apps-git-user string       Your git provider user name [APPS_GIT_USER] (not required in GitHub)
      --apps-repo string           Repository URL [APPS_GIT_REPO]
      --context string             The name of the kubeconfig context to use
      --dest-namespace string      K8s target namespace (overrides the namespace specified in the kustomization.yaml)
      --dest-server string         K8s cluster URL (e.g. https://kubernetes.default.svc) (default "https://kubernetes.default.svc")
      --exclude string             Optional glob for files to exclude
  -t, --git-token string           Your git provider api token [GIT_TOKEN]
  -u, --git-user string            Your git provider user name [GIT_USER] (not required in GitHub)
  -h, --help                       help for create
      --include string             Optional glob for files to include
      --installation-mode string   One of: normal|flat. If flat, will commit the application manifests (after running kustomize build), otherwise will commit the kustomization.yaml (default "normal")
      --kubeconfig string          Path to the kubeconfig file to use for CLI requests.
      --labels stringToString      Optional labels that will be set on the Application resource. (e.g. "{{ placeholder }}=my-org" (default [])
  -n, --namespace string           If present, the namespace scope for this CLI request
  -p, --project string             Project name
      --repo string                Repository URL [GIT_REPO]
      --request-timeout string     The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
      --type string                The application type (kustomize|dir)
  -b, --upsert-branch              If true will try to checkout the specified branch and create it if it doesn't exist
      --wait-timeout duration      If not '0s', will try to connect to the cluster and wait until the application is in 'Synced' status for the specified timeout period
```

### SEE ALSO

* [argocd-autopilot application](argocd-autopilot_application.md)	 - Manage applications


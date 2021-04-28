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
    
        --token <token> --repo <repo_url>
        
# Create a new application from kustomization in a remote repository
    
    argocd-autopilot app create <new_app_name> --app github.com/some_org/some_repo/manifests?ref=v1.2.3 --project project_name

```

### Options

```
      --app string                 The application specifier (e.g. argocd@v1.0.2)
      --dest-namespace string      K8s target namespace (overrides the namespace specified in the kustomization.yaml)
      --dest-server string         K8s cluster URL (e.g. https://kubernetes.default.svc) (default "https://kubernetes.default.svc")
  -t, --git-token string           Your git provider api token [GIT_TOKEN]
  -h, --help                       help for create
      --installation-mode string   One of: normal|flat. If flat, will commit the application manifests (after running kustomize build), otherwise will commit the kustomization.yaml (default "normal")
      --installation-path string   The path where we of the installation files (defaults to the root of the repository [GIT_INSTALLATION_PATH]
  -p, --project string             Project name
      --repo string                Repository URL [GIT_REPO]
      --revision string            Repository branch, tag or commit hash (defaults to HEAD)
```

### SEE ALSO

* [argocd-autopilot application](argocd-autopilot_application.md)	 - Manage applications


## argocd-autopilot repo create

Create a new gitops repository

```
argocd-autopilot repo create [flags]
```

### Examples

```

# To run this command you need to create a personal access token for your git provider
# and provide it using:
    
    export GIT_TOKEN=<token>

# or with the flag:
    
    --token <token>

# Create a new gitops repository on github
    
    argocd-autopilot repo create --owner foo --name bar --token abc123

# Create a public gitops repository on github
    
    argocd-autopilot repo create --owner foo --name bar --token abc123 --public

```

### Options

```
  -t, --git-token string   Your git provider api token [GIT_TOKEN]
  -h, --help               help for create
      --host string        The git provider address (for on-premise git providers)
  -n, --name string        The name of the repository
  -o, --owner string       The name of the owner or organiaion
  -p, --provider string    The git provider, one of: github (default "github")
      --public             If true, will create the repository as public (default is false)
```

### SEE ALSO

* [argocd-autopilot repo](argocd-autopilot_repo.md)	 - Manage gitops repositories


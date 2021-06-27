## argocd-autopilot project delete

Delete a project and all of its applications

```
argocd-autopilot project delete [PROJECT_NAME] [flags]
```

### Examples

```

# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:
    
        export GIT_TOKEN=<token>
        export GIT_REPO=<repo_url>

# or with the flags:
    
        --token <token> --repo <repo_url>
        
# Delete a project
    
    argocd-autopilot project delete <project_name>

```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
  -t, --git-token string   Your git provider api token [GIT_TOKEN]
      --provider string    The git provider, one of: github
      --repo string        Repository URL [GIT_REPO]
```

### SEE ALSO

* [argocd-autopilot project](argocd-autopilot_project.md)	 - Manage projects


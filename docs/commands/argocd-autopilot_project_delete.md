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
      --git-server-crt string   Git Server certificate file
  -t, --git-token string        Your git provider api token [GIT_TOKEN]
  -u, --git-user string         Your git provider user name [GIT_USER] (not required in GitHub)
  -h, --help                    help for delete
      --repo string             Repository URL [GIT_REPO]
  -b, --upsert-branch           If true will try to checkout the specified branch and create it if it doesn't exist
```

### SEE ALSO

* [argocd-autopilot project](argocd-autopilot_project.md)	 - Manage projects


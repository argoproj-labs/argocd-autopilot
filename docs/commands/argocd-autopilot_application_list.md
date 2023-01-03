## argocd-autopilot application list

List all applications in a project

```
argocd-autopilot application list [PROJECT_NAME] [flags]
```

### Examples

```

# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:

        export GIT_TOKEN=<token>
        export GIT_REPO=<repo_url>

# or with the flags:

        --git-token <token> --repo <repo_url>

# Get list of installed applications in a specifc project

    argocd-autopilot app list <project_name>

```

### Options

```
      --git-server-crt string   Git Server certificate file
  -t, --git-token string        Your git provider api token [GIT_TOKEN]
  -u, --git-user string         Your git provider user name [GIT_USER] (not required in GitHub)
  -h, --help                    help for list
      --repo string             Repository URL [GIT_REPO]
```

### SEE ALSO

* [argocd-autopilot application](argocd-autopilot_application.md)	 - Manage applications


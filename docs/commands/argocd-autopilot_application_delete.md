## argocd-autopilot application delete

Delete an application from a project

```
argocd-autopilot application delete [APP_NAME] [flags]
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

    argocd-autopilot app delete <app_name> --project <project_name>

```

### Options

```
      --git-server-crt string   Git Server certificate file
  -t, --git-token string        Your git provider api token [GIT_TOKEN]
  -u, --git-user string         Your git provider user name [GIT_USER] (not required in GitHub)
  -g, --global                  global
  -h, --help                    help for delete
  -p, --project string          Project name
      --repo string             Repository URL [GIT_REPO]
  -b, --upsert-branch           If true will try to checkout the specified branch and create it if it doesn't exist
```

### SEE ALSO

* [argocd-autopilot application](argocd-autopilot_application.md)	 - Manage applications


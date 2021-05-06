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
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -t, --git-token string           Your git provider api token [GIT_TOKEN]
      --installation-path string   The path where we of the installation files (defaults to the root of the repository [GIT_INSTALLATION_PATH]
  -p, --project string             Project name
      --repo string                Repository URL [GIT_REPO]
      --revision string            Repository branch, tag or commit hash (defaults to HEAD)
```

### SEE ALSO

* [argocd-autopilot application](argocd-autopilot_application.md)	 - Manage applications


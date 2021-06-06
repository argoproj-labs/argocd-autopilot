## argocd-autopilot project list

Lists all the projects on a git repository

```
argocd-autopilot project list  [flags]
```

### Examples

```

# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:

        export GIT_TOKEN=<token>
        export GIT_REPO=<repo_url>

# or with the flags:

        --git-token <token> --repo <repo_url>

# Lists projects

    argocd-autopilot project list

```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -t, --git-token string   Your git provider api token [GIT_TOKEN]
      --repo string        Repository URL [GIT_REPO]
```

### SEE ALSO

* [argocd-autopilot project](argocd-autopilot_project.md)	 - Manage projects


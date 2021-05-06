## argocd-autopilot project

Manage projects

```
argocd-autopilot project [flags]
```

### Options

```
  -t, --git-token string           Your git provider api token [GIT_TOKEN]
  -h, --help                       help for project
      --installation-path string   The path where we of the installation files (defaults to the root of the repository [GIT_INSTALLATION_PATH]
  -p, --project string             Project name
      --repo string                Repository URL [GIT_REPO]
      --revision string            Repository branch, tag or commit hash (defaults to HEAD)
```

### SEE ALSO

* [argocd-autopilot](argocd-autopilot.md)	 - argocd-autopilot is used for installing and managing argo-cd installations and argo-cd
applications using gitops
* [argocd-autopilot project create](argocd-autopilot_project_create.md)	 - Create a new project
* [argocd-autopilot project delete](argocd-autopilot_project_delete.md)	 - Delete a project and all of its applications
* [argocd-autopilot project list](argocd-autopilot_project_list.md)	 - Lists all the projects on a git repository


## argocd-autopilot application

Manage applications

```
argocd-autopilot application [flags]
```

### Options

```
  -t, --git-token string           Your git provider api token [GIT_TOKEN]
  -h, --help                       help for application
      --installation-path string   The path where we of the installation files (defaults to the root of the repository [GIT_INSTALLATION_PATH]
      --repo string                Repository URL [GIT_REPO]
      --revision string            Repository branch, tag or commit hash (defaults to HEAD)
```

### SEE ALSO

* [argocd-autopilot](argocd-autopilot.md)	 - argocd-autopilot is used for installing and managing argo-cd installations and argo-cd
applications using gitops
* [argocd-autopilot application create](argocd-autopilot_application_create.md)	 - Create an application in a specific project
* [argocd-autopilot application delete](argocd-autopilot_application_delete.md)	 - Delete an application from a project
* [argocd-autopilot application list](argocd-autopilot_application_list.md)	 - List all applications in a project


## argocd-autopilot

argocd-autopilot is used for installing and managing argo-cd installations and argo-cd
applications using gitops

### Synopsis

argocd-autopilot is used for installing and managing argo-cd installations and argo-cd
applications using gitops.
        
Most of the commands in this CLI require you to specify a personal access token
for your git provider. This token is used to authenticate with your git provider
when performing operations on the gitops repository, such as cloning it and
pushing changes to it.

It is recommended that you export the $GIT_TOKEN and $GIT_REPO environment
variables in advanced to simplify the use of those commands.


```
argocd-autopilot [flags]
```

### Options

```
  -h, --help   help for argocd-autopilot
```

### SEE ALSO

* [argocd-autopilot application](argocd-autopilot_application.md)	 - Manage applications
* [argocd-autopilot project](argocd-autopilot_project.md)	 - Manage projects
* [argocd-autopilot repo](argocd-autopilot_repo.md)	 - Manage gitops repositories
* [argocd-autopilot version](argocd-autopilot_version.md)	 - Show cli version


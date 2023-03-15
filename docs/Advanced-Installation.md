# Advanced Installation Options


### Bootstrap under a specific path
If you want the autopilot managed folder structure to reside under some sub-folder in your repository, you can also export the following env variable:
```
export GIT_REPO=https://github.com/owner/name.git/some/relative/path
```
You should see the `bootstrap/` directory under `/some/relative/path/bootstrap`.

### Bootstrap on a specific branch
If you want to use a specific branch for your GitOps repository operations, you can use the `ref` query parameter:
```
export GIT_REPO=https://github.com/owner/name?ref=gitops_branch
```

!!! note
    When running commands that commit or write to the repository, the value of `ref` can only be a branch.


!!! tip
    When running commands that commit or write to the repository you may also specify the `-b`, this would create the branch specified in `ref` if it doesn't exist. 

    Note that when doing so the new branch would be created from the default branch.



### High Availability
You can bootstrap Argo CD in high-availability mode using the [App Specifier](App-Specifier/):
```
argocd-autopilot repo bootstrap --app https://github.com/argoproj-labs/argocd-autopilot/manifests/ha
```

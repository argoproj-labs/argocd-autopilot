#### Recovering From an Existing Repository
If, for any reason, something happens to your argo-cd installation, you can recover from an existing repository using the `--recover` flag.

This should re-apply Argo-CD to the cluster, then create the `autopilot-bootstrap` application, which will restore all of the other applications in your repository.



```
export GIT_REPO=https://github.com/owner/installation-repo
export GIT_TOKEN=xxx

argocd-autopilot repo bootstrap --recover
```

#### Apply Argo-CD Manifests From Existing Repository
In some cases where you made [some modifications](./Modifying-Argo-CD.md) to your Argo-CD, you probably want to apply the modified Argo-CD manifests from your repository instead of new ones. You can easily do that with the `--app` flag.

For example:
```
export GIT_REPO=https://github.com/owner/installation-repo
export GIT_TOKEN=xxx

argocd-autopilot repo bootstrap --recover --app "${GIT_REPO}.git/bootstrap/argo-cd"
```

This is using the [app specifier](./App-Specifier.md) flag to tell autopilot that the Argo-CD manifests should be generated from `/bootstrap/argo-cd/kustomization.yaml`.

!!! note
    If you used a different path or branch for your autopilot installation your app specifier would look different

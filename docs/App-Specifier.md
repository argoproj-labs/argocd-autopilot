# The Application Specifier

The application specifier is a string denoting the entrypoint to the application that you want to create. You specify it when using the `--app`
flag in the `app create` and `repo bootstrap` commands.

## Structure
Lets look at the following example of adding argo workflows v3.0.7 to project `prod` to better understand the structure of the application specifier:
```bash
argocd-autopilot app create workflows --app "github.com/argoproj/argo-workflows/manifests/cluster-install?ref=v3.0.7" --project prod
```
In this example the app specifier is: `github.com/argoproj/argo-workflows/manifests/cluster-install?ref=v3.0.7`, which is composed of three parts:

1. `github.com/argoproj/argo-workflows`: The repository
2. `manifests/cluster-install`: The path inside the repository to the directory containing the base `kustomization.yaml`
3. `?ref=v3.0.7`: The git ref to use, in this case, the tag `v3.0.7`

!!! note
    The `ref` that will be used to get the application manifests is calculated using the following logic:

      1. If not specified - uses the HEAD of the main branch of the repository
      2. If there is a commit with the same SHA use this commit
      3. Looks for a tag with the same name
      4. Looks for a branch with the same name

## Application Type Inference
By default, `argocd-autopilot` will try to automatically infer the correct application type from the supported [application types](https://argoproj.github.io/argo-cd/user-guide/application_sources/#tools) (currently only kustomize and directory types are supported). To do that it would try to clone the repository, checkout the correct ref, and look at the specified path for the following:

1. If there is a `kustomization.yaml` - the infered application type is `kustomize`
2. Else - the infered application type is `directory`

!!! tip
    If you don't want `argocd-autopilot` to infer the type automatically, you can specify the application type yourself using the `--type` flag.

## Local Application Path
If the application specifier is a path to a local directory on your machine, `argocd autopilot` will automatically detect that and use `flat` installation mode, meaning it would build all of the manifests and write them into one `install.yaml` file, which would be required by a base `kustomization.yaml`.

For example:
```
argocd-autopilot app create someapp --app ./path/to/kustomization/dir --project dev
```
Assuming the file `./path/to/kustomization/dir/kustomization.yaml` exists, `argocd-autopilot` will run `kustomize build`, then commit the resulting manifests to the gitops repository under: `apps/someapp/base/install.yaml`, with the base kustomization, located at `apps/someapp/base/kustomization.yaml`, requiring it.
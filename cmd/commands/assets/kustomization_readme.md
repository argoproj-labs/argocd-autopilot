# Kustomizations
This directory contains all of the applications you installed by using:
```bash
argocd-autopilot app create <APP_NAME> --app <APP_SPECIFIER> -p <PROJECT_NAME>
```

Every application you install has <u>exactly one</u>: `kustomize/<APP_NAME>/base/kustomization.yaml` and one or more `kustomize/<APP_NAME>/overlays/<PROJECT_NAME>/kustomization.yaml` files.

The `kustomize/<APP_NAME>/base/kustomization.yaml` file is created the first time you create the application. The `kustomize/<APP_NAME>/overlays/<PROJECT_NAME>/kustomization.yaml` is created for each project you install this application on. So all overlays of the same application are using the same base `kustomization.yaml`.

## Example:
Try running the following command:
```bash
argocd-autopilot app create hello-world --app github.com/argoproj-labs/argocd-autopilot/examples/demo-app/ -p <PROJECT_NAME>
```
###### * If you did not create a project yet take a look at: [creating a project](https://argocd-autopilot.readthedocs.io/en/stable/Getting-Started/#add-a-project-and-an-application).
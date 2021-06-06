# Apps
This directory contains all of the applications you installed by using:
```bash
argocd-autopilot app create <APP_NAME> --app <APP_SPECIFIER> -p <PROJECT_NAME>
```

## Application Types
> If you don't specify the application `--type` argocd-autopilot will try to clone the source repository and infer the application type [automatically](https://argoproj.github.io/argo-cd/user-guide/tool_detection/#tool-detection)

* ### Directory application
  Such an application references a specific directory at a given repo URL, path and revision. It will be persisted in the GitOps Repository as a single file at `apps/<APP_NAME>/<PROJECT_NAME>/config.json`.  
  #### Example:  
  ```bash
  argocd-autopilot app create dir-example --app github.com/argoproj-labs/argocd-autopilot/examples/demo-dir/ -p <PROJECT_NAME> --type dir
  ```

* ### Kustomize application
  A Kustomize application will have <u>exactly one</u>: `apps/<APP_NAME>/base/kustomization.yaml` file, and one or more `apps/<APP_NAME>/overlays/<PROJECT_NAME>/` folders.

  The `apps/<APP_NAME>/base/kustomization.yaml` file is created the first time you create the application. The `apps/<APP_NAME>/overlays/<PROJECT_NAME>/` folder is created for each project you install this application on. So all overlays of the same application are using the same base `kustomization.yaml`.
  #### Example:
  Try running the following command:
  ```bash
  argocd-autopilot app create hello-world --app github.com/argoproj-labs/argocd-autopilot/examples/demo-app/ -p <PROJECT_NAME> --type kustomize
  ```

###### * If you did not create a project yet take a look at: [creating a project](https://argocd-autopilot.readthedocs.io/en/stable/Getting-Started/#add-a-project-and-an-application).
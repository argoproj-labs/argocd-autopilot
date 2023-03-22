# Getting Started

This guide assumes you are familiar with Argo CD and its basic concepts. See the [Argo CD documentation](https://argoproj.github.io/argo-cd/core_concepts/) for more information.

## Before you Begin 
### Requirements

* Installed [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`)

### Export a Git Token
```
export GIT_TOKEN=ghp_PcZ...IP0
```

Make sure you export a [valid token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token) with the required scopes.
![Github token](assets/github_token.png)

### Export Clone URL
You can use any clone URL to a valid git repo, provided that the token you supplied earlier will allow cloning from, and pushing to it.
If the repository does not exist, bootstrapping it will also create it as a private repository.
```
export GIT_REPO=https://github.com/owner/name
```

#### Using a Specific git Provider
You can add the `--provider` flag to the `repo bootstrap` command, to enforce using a specific provider when creating a new repository. If the value is not supplied, the code will attempt to infer it from the clone URL.

Autopilot currently supports a variety of git providers, you should check if yours is currently supported [here](./Git-Providers.md).


## Bootstrap Argo-CD 

Now that you have exported the `GIT_TOKEN` and `GIT_REPO` environment variables you can run the bootstrap command:
```
argocd-autopilot repo bootstrap
```

This command will install Argo-CD on your current Kubernetes context in the `argocd` namespace. You might need to wait a few minutes while the required images are being pulled.

After Argo-CD is up and running autopilot will push the installation manifests to the installation repository and create the `autopilot-bootstrap` application in the cluster. This will in turn deploy the `argo-cd` application, making Argo-CD manage itself, completing the bootstrap process.

Before the bootstrap command is finished it will print out the initial Argo-CD admin password, as well as the command to run to enable port-forwarding:
```
INFO argocd initialized. password: pfrDVRJZtHYZKzBv 
INFO run:

    kubectl port-forward -n argocd svc/argocd-server 8080:80
```
<sub>(Your initial password will be different)</sub>

Execute the port forward command, and browse to `http://localhost:8080`. You can log in with user: `admin`, and the password from the previous step.

Your initial Argo CD should have the following applications:

* `autopilot-bootstrap` - References the `bootstrap` directory in the GitOps repository, and manages the other 2 applications
* `argo-cd` - References the `bootstrap/argo-cd` folder, and manages the Argo CD deployment itself (including Argo CD ApplicationSet)
* `root` - References the `projects` directiry in the repo. The folder contains only an empty `DUMMY` file after the bootstrap command, so no projects will be created

<!-- FIXME: Screenshot is outdated; missing the `cluster-resources-in-cluster` Application introduced with #79. -->
![Step 1](assets/getting_started_1.png)


## Create a Project
Projects provide a way to logically group applications and easily control things such as defaults and restrictions.

Projects may also be used to deploy applications to different kubernetes clusters.

To create your first project run the following command:
```
argocd-autopilot project create testing
```
This will create the `testing` [AppProject](https://argo-cd.readthedocs.io/en/stable/user-guide/projects/) and [ApplicationSet](https://argo-cd.readthedocs.io/en/stable/user-guide/application-set/). You should see that it was pushed to your installation repository under `/projects/testing.yaml`

## Add an Application
Now that you have your first project, Argo-CD applications can be added to it and deployed with a simple command:
```
argocd-autopilot app create hello-world --app github.com/argoproj-labs/argocd-autopilot/examples/demo-app/ -p testing --wait-timeout 2m
```
<sub>* notice the trailing slash in the URL</sub>

This will create an application with the name `hello-world` and add it to the `testing` project.

After the application is created, and Argo-CD has finished its sync cycle, your new `testing` project will appear under the `root` application:

![Step 2](assets/getting_started_2.png)

And the `hello-world` application will also be deployed to the cluster:

![Step 3](assets/getting_started_3.png)

## Uninstalling
The following command will clear your GitOps repository of related files, and your Kubernetes cluster from any autopilot related resources (including Argo-CD itself)
```
argocd-autopilot repo uninstall
```

## Advanced Use Cases
For more advanced use-case, which includes deploying to multiple environments, you can go through this [argocd-autopilot deep dive](https://codefresh.io/about-gitops/launching-argo-cd-autopilot-opinionated-way-manage-applications-across-environments-using-gitops-scale/) blog post.

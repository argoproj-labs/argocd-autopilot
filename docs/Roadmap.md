# Roadmap

### Repo Upgrade and Uninstall
* Support a clear flow to [upgrade](https://github.com/argoproj-labs/argocd-autopilot/issues/45) Argo CD

### App Upgrade and Delete
* Support a clear flow to [upgrade](https://github.com/argoproj-labs/argocd-autopilot/issues/44) an app

### Working with and Storing Secrets 
* [Git token should also be maintained in a GitOps approach](https://github.com/argoproj-labs/argocd-autopilot/issues/25) 
* Addition of destination clusters should be maintained in a GitOps approach
* supporting automatic integration with external secret stores
* provide out of the box secret store solution in case of not bringing an existing one

### Promote Feature
* Provide a clear way to automate the process of promoting changes between one to another environment
* Provide a clear way to automate the process of promoting changes to all environments after verification on a single environment

### Multiple Argo CD installations Targeting Specific Environments
In some organizations there is a need to separate production from other testing environments because of regulations and networking restrictions.

That been said, in most cases you will still want to have a single GitOps repository that contains all your environments.

So for example you can have an Argo CD installation controlling your production but another Argo CD installation that will target all the rest of environments.

### Other Templating Choices Besides Kustomize
* Support Helm 

### Additional Git Providers Support for Repo Create
* support [Bitbucket](https://github.com/argoproj-labs/argocd-autopilot/issues/7)
* support [GitLab](https://github.com/argoproj-labs/argocd-autopilot/issues/6)

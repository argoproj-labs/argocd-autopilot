# argocd-autopilot
[![codecov](https://codecov.io/gh/codefresh-io/cf-argo/branch/main/graph/badge.svg?token=R64AZI8NUW)](https://codecov.io/gh/codefresh-io/cf-argo)
## Overview:
The argocd-autopilot utilizes the gitops pattern in order to control the install,uninstall and upgrade flows for kustomize based installations.
The argocd-autopilot cli modifies a git repository while leverging the Argo CD apps patttern

## Architecture:
### Bootstrap:
The argocd-autopilot bootstrap command pushs the apps manifests into a git repository which will be later used to contorl the gitops lifecycle.
Later the  argocd-autopilot install Argo CD on kubernetes cluster which monitors the repository for changes. Now one cad add projects and apps in a gitops approach

## Usage:

### Creating a new project
```
~ argocd-autopilot project create [PROJECT] [flags]
This command will create a new project which will later can add applications to it

Usage:
  argocd-autopilot project create [PROJECT] [flags]

--auth-token string                  Authentication token
      --aws-cluster-name string            AWS Cluster name if set then aws cli eks token command will be used to access cluster
      --aws-role-arn string                Optional AWS role arn. If set then AWS IAM Authenticator assumes a role to perform cluster operations instead of the default AWS credential provider chain.
      --client-crt string                  Client certificate file
      --client-crt-key string              Client certificate key file
      --config string                      Path to Argo CD config (default "/home/user/.argocd/config")
      --dest-kube-context string           The default destination kubernetes context for applications in this project
      --dry-run                            If true, print manifests instead of applying them to the cluster (nothing will be commited to git)
      --exec-command string                Command to run to provide client credentials to the cluster. You may need to build a custom ArgoCD image to ensure the command is available at runtime.
      --exec-command-api-version string    Preferred input version of the ExecInfo for the --exec-command executable
      --exec-command-args stringArray      Arguments to supply to the --exec-command executable
      --exec-command-env stringToString    Environment vars to set when running the --exec-command executable (default [])
      --exec-command-install-hint string   Text shown to the user when the --exec-command executable doesn't seem to be present
  -t, --git-token string                   Your git provider api token [GIT_TOKEN]
      --grpc-web                           Enables gRPC-web protocol. Useful if Argo CD server is behind proxy which does not support HTTP2.
      --grpc-web-root-path string          Enables gRPC-web protocol. Useful if Argo CD server is behind proxy which does not support HTTP2. Set web root.
  -H, --header strings                     Sets additional header to all requests made by Argo CD CLI. (Can be repeated multiple times to add multiple headers, also supports comma separated headers)
  -h, --help                               help for create
      --in-cluster                         Indicates Argo CD resides inside this cluster and should connect using the internal k8s hostname (kubernetes.default.svc)
      --insecure                           Skip server certificate and domain verification
      --installation-path string           The path where we of the installation files (defaults to the root of the repository [GIT_INSTALLATION_PATH]
      --name string                        Overwrite the cluster name
      --plaintext                          Disable TLS
      --port-forward                       Connect to a random argocd-server port using port forwarding
      --port-forward-namespace string      Namespace name which should be used for port forwarding
      --repo string                        Repository URL [GIT_REPO]
      --revision string                    Repository branch, tag or commit hash (defaults to HEAD)
      --server string                      Argo CD server address
      --server-crt string                  Server certificate file
      --service-account string             System namespace service account to use for kubernetes resource management. If not set then default "argocd-manager" SA will be created
      --shard int                          Cluster shard number; inferred from hostname if not set (default -1)
      --system-namespace string            Use different system namespace (default "kube-system")
      --upsert                             Override an existing cluster with the same name even if the spec differs
```
### Creating a new application

~ argocd-autopilot application create [APP_NAME] [flags]
This command will create a new application under a project in the git reposiotry 

Usage:
  argocd-autopilot application create [APP_NAME] [flags]

    --app string                 The application specifier (e.g. argocd@v1.0.2)
      --dest-namespace string      K8s target namespace (overrides the namespace specified in the kustomization.yaml)
      --dest-server string         K8s cluster URL (e.g. https://kubernetes.default.svc) (default "https://kubernetes.default.svc")
  -t, --git-token string           Your git provider api token [GIT_TOKEN]
  -h, --help                       help for create
      --installation-mode string   One of: normal|flat. If flat, will commit the application manifests (after running kustomize build), otherwise will commit the kustomization.yaml (default "normal")
      --installation-path string   The path where we of the installation files (defaults to the root of the repository [GIT_INSTALLATION_PATH]
  -p, --project string             Project name
      --repo string                Repository URL [GIT_REPO]
      --revision string            Repository branch, tag or commit hash (defaults to HEAD)


## Development

### Building from Source:
To build a binary from the source code, make sure:
* you have `go >=1.16` installed.
* and that the `GOPATH` environment variable is set.


Then run:
* `make` to build the binary to `./dist/`  


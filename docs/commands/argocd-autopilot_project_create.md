## argocd-autopilot project create

Create a new project

```
argocd-autopilot project create [PROJECT] [flags]
```

### Examples

```

# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:

        export GIT_TOKEN=<token>
        export GIT_REPO=<repo_url>

# or with the flags:

        --git-token <token> --repo <repo_url>

# Create a new project

    argocd-autopilot project create <PROJECT_NAME>

```

### Options

```
      --annotation stringArray             Set metadata annotations (e.g. --annotation key=value)
      --auth-token string                  Authentication token
      --aws-cluster-name string            AWS Cluster name if set then aws cli eks token command will be used to access cluster
      --aws-role-arn string                Optional AWS role arn. If set then AWS IAM Authenticator assumes a role to perform cluster operations instead of the default AWS credential provider chain.
      --client-crt string                  Client certificate file
      --client-crt-key string              Client certificate key file
      --cluster-resources                  Indicates if cluster level resources should be managed. The setting is used only if list of managed namespaces is not empty.
      --config string                      Path to Argo CD config (default "/home/user/.config/argocd/config")
      --core                               If set to true then CLI talks directly to Kubernetes instead of talking to Argo CD API server
      --dest-kube-context string           The default destination kubernetes context for applications in this project (will be ignored if --dest-kube-server is supplied)
      --dest-server string                 The default destination kubernetes server for applications in this project
      --dry-run                            If true, print manifests instead of applying them to the cluster (nothing will be commited to git)
      --exec-command string                Command to run to provide client credentials to the cluster. You may need to build a custom ArgoCD image to ensure the command is available at runtime.
      --exec-command-api-version string    Preferred input version of the ExecInfo for the --exec-command executable
      --exec-command-args stringArray      Arguments to supply to the --exec-command executable
      --exec-command-env stringToString    Environment vars to set when running the --exec-command executable (default [])
      --exec-command-install-hint string   Text shown to the user when the --exec-command executable doesn't seem to be present
  -t, --git-token string                   Your git provider api token [GIT_TOKEN]
  -u, --git-user string                    Your git provider user name [GIT_USER] (not required in GitHub)
      --grpc-web                           Enables gRPC-web protocol. Useful if Argo CD server is behind proxy which does not support HTTP2.
      --grpc-web-root-path string          Enables gRPC-web protocol. Useful if Argo CD server is behind proxy which does not support HTTP2. Set web root.
  -H, --header strings                     Sets additional header to all requests made by Argo CD CLI. (Can be repeated multiple times to add multiple headers, also supports comma separated headers)
  -h, --help                               help for create
      --http-retry-max int                 Maximum number of retries to establish http connection to Argo CD server
      --in-cluster                         Indicates Argo CD resides inside this cluster and should connect using the internal k8s hostname (kubernetes.default.svc)
      --insecure                           Skip server certificate and domain verification
      --label stringArray                  Set metadata labels (e.g. --label key=value)
      --labels stringToString              Optional labels that will be set on the Application resource. (e.g. "app.kubernetes.io/managed-by={{ placeholder }}" (default [])
      --name string                        Overwrite the cluster name
      --plaintext                          Disable TLS
      --port-forward                       Connect to a random argocd-server port using port forwarding
      --port-forward-namespace string      Namespace name which should be used for port forwarding
      --project string                     project of the cluster
      --repo string                        Repository URL [GIT_REPO]
      --server string                      Argo CD server address
      --server-crt string                  Server certificate file
      --service-account string             System namespace service account to use for kubernetes resource management. If not set then default "argocd-manager" SA will be created
      --shard int                          Cluster shard number; inferred from hostname if not set (default -1)
      --system-namespace string            Use different system namespace (default "kube-system")
      --upsert                             Override an existing cluster with the same name even if the spec differs
  -b, --upsert-branch                      If true will try to checkout the specified branch and create it if it doesn't exist
  -y, --yes                                Skip explicit confirmation
```

### SEE ALSO

* [argocd-autopilot project](argocd-autopilot_project.md)	 - Manage projects


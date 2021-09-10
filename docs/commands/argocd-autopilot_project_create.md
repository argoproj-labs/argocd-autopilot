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
      --as string                          Username to impersonate for the operation
      --as-group stringArray               Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --auth-token string                  Authentication token
      --aws-cluster-name string            AWS Cluster name if set then aws cli eks token command will be used to access cluster
      --aws-role-arn string                Optional AWS role arn. If set then AWS IAM Authenticator assumes a role to perform cluster operations instead of the default AWS credential provider chain.
      --certificate-authority string       Path to a cert file for the certificate authority
      --client-certificate string          Path to a client certificate file for TLS
      --client-crt string                  Client certificate file
      --client-crt-key string              Client certificate key file
      --client-key string                  Path to a client key file for TLS
      --cluster string                     The name of the kubeconfig cluster to use
      --cluster-resources                  Indicates if cluster level resources should be managed. The setting is used only if list of managed namespaces is not empty.
      --config string                      Path to Argo CD config (default "/home/user/.argocd/config")
      --context string                     The name of the kubeconfig context to use
      --core                               If set to true then CLI talks directly to Kubernetes instead of talking to Argo CD API server
      --dest-kube-context string           The default destination kubernetes context for applications in this project
      --dry-run                            If true, print manifests instead of applying them to the cluster (nothing will be commited to git)
      --exec-command string                Command to run to provide client credentials to the cluster. You may need to build a custom ArgoCD image to ensure the command is available at runtime.
      --exec-command-api-version string    Preferred input version of the ExecInfo for the --exec-command executable
      --exec-command-args stringArray      Arguments to supply to the --exec-command executable
      --exec-command-env stringToString    Environment vars to set when running the --exec-command executable (default [])
      --exec-command-install-hint string   Text shown to the user when the --exec-command executable doesn't seem to be present
      --grpc-web                           Enables gRPC-web protocol. Useful if Argo CD server is behind proxy which does not support HTTP2.
      --grpc-web-root-path string          Enables gRPC-web protocol. Useful if Argo CD server is behind proxy which does not support HTTP2. Set web root.
  -H, --header strings                     Sets additional header to all requests made by Argo CD CLI. (Can be repeated multiple times to add multiple headers, also supports comma separated headers)
  -h, --help                               help for create
      --http-retry-max int                 Maximum number of retries to establish http connection to Argo CD server
      --in-cluster                         Indicates Argo CD resides inside this cluster and should connect using the internal k8s hostname (kubernetes.default.svc)
      --insecure                           Skip server certificate and domain verification
      --insecure-skip-tls-verify           If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --name string                        Overwrite the cluster name
      --password string                    Password for basic authentication to the API server
      --plaintext                          Disable TLS
      --port-forward                       Connect to a random argocd-server port using port forwarding
      --port-forward-namespace string      Namespace name which should be used for port forwarding
      --request-timeout string             The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
      --server string                      The address and port of the Kubernetes API server
      --server-crt string                  Server certificate file
      --service-account string             System namespace service account to use for kubernetes resource management. If not set then default "argocd-manager" SA will be created
      --shard int                          Cluster shard number; inferred from hostname if not set (default -1)
      --system-namespace string            Use different system namespace (default "kube-system")
      --tls-server-name string             If provided, this name will be used to validate server certificate. If this is not provided, hostname used to contact the server is used.
      --token string                       Bearer token for authentication to the API server
      --upsert                             Override an existing cluster with the same name even if the spec differs
      --user string                        The name of the kubeconfig user to use
      --username string                    Username for basic authentication to the API server
  -y, --yes                                Skip explicit confirmation
```

### Options inherited from parent commands

```
  -t, --git-token string   Your git provider api token [GIT_TOKEN]
  -u, --git-user string    Your git provider user name [GIT_USER] (not required in GitHub)
      --repo string        Repository URL [GIT_REPO]
```

### SEE ALSO

* [argocd-autopilot project](argocd-autopilot_project.md)	 - Manage projects


# cf-argo
codefresh argo cli

## Usage:
```
~ cf-argo install --help

This command will create a new git repository that manages an Argo Enterprise solution using Argo-CD with gitops.

Usage:
  cf-argo install [flags]

Flags:
      --dry-run               when true, the command will have no side effects, and will only output the manifests to stdout
      --env-name string       name of the Argo Enterprise environment to create (default "production")
      --git-token string      git token which will be used by argo-cd to create the gitops repository
  -h, --help                  help for install
      --kube-context string   name of the kubeconfig context to use (default: current context)
      --kubeconfig string     path to the kubeconfig file [KUBECONFIG] (default: ~/.kube/config)
      --repo-url string       the gitops repository url. If it does not exist we will try to create it for you [REPO_URL]

Global Flags:
      --log-format string   set the log format: "text", "json" (defaults to text) (default "text")
      --log-level string    set the log level, e.g. "debug", "info", "warn", "error" (default "info")
```

## Development

### Building from Source:
To build a binary from the source code, make sure:
* you have `go >=1.15` installed.
* and that the `GOPATH` environment variable is set.


Then run:
* `make` to build the binary to `./dist/`  


or 
* `make install` to make it available as `cf-argo` in the `PATH`
### Linting:
We are using https://github.com/golangci/golangci-lint as our linter, you can integrate golangci-lint with the following IDEs:
* vscode: make sure `GOPATH` is setup correctly and run `make lint` this will download `golangci-lint` if it was not already installed on your machine. Then add the following to your `settings.json`:
```
"go.lintTool": "golangci-lint",
"go.lintFlags": [
    "--fast"
],
```

### Bumping template repository version:
By default the cli will use the repository set in the makefile as `BASE_GIT_URL` as the base template repository, when there is a new version of the template repository, you need to release a new version of the installer and bump the version of the `BASE_GIT_URL` in the makefile. The base repository can also be controlled with the hidden flag `--base-repo`.

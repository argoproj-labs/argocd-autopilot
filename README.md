# cf-argo
codefresh argo cli

## Development
### Linting
we are using https://github.com/golangci/golangci-lint as our linter, you can integrate golangci-lint with the following IDEs:
* vscode: make sure `GOPATH` is setup correctly and run `make lint` this will download `golangci-lint` if it was not already installed on your machine. Then add the following to your `settings.json`:
```
"go.lintTool": "golangci-lint",
"go.lintFlags": [
    "--fix"
],
```

### Installation Steps
1. [x] Clone
2. [ ] Template Substitution
3. [x] kustomize build
4. [x] git
   1. [x] init
   2. [x] add
   3. [x] commit
   4. [x] create Remote repo
   5. [x] push
5. [ ] Add config-map + secret of token
6. [ ] apply yaml from earlier
7. [ ] Create seaeled-secret from secret
8. [ ] add/commit/push
9. [ ] party

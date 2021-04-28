# Development
This guide is meant for developers who want to contribute or debug `argocd-autopilot`.

### Adding a new feature:
1. Fork the repository.
2. Clone it and add the upstream remote with: `git remote add upstream https://github.com/argoproj-labs/argocd-autopilot.git`.
3. Run `make local` to build the project.
4. Add your feature, run `make pre-commit`, then commit.
5. Run `make pre-push`.
6. Push the changes to the remote branch and create a new PR: `git push --set-upstream upstream <remote-branch-name>`.
7. If you need to get changes from the upstream repo, run: `git pull upstream main`.

### Releasing a new version:
1. Checkout to a release branch: `v0.X.X`.
2. Change the `VERSION` in the Makefile to match the new version.
3. Add what you need to `./docs/releases/release_notes.md`.
4. Create a new PR to the `main` branch.
5. After CI is green, add a `/release` comment to the PR to trigger the release pipeline (maintainers only).
6. After Release build is finished you can merge back to `main`.

### Using pre-commit:
With pre-commit installed and properly set-up, both the pre-commit and pre-push hooks will run automatically.
1. Install [pre-commit](https://pre-commit.com/#install) on your machine
2. Install the hooks in the repo folder: `pre-commit install -t pre-commit -t pre-push`
3. Enjoy


### Linting:
We are using https://github.com/golangci/golangci-lint as our linter, you can integrate golangci-lint with the following IDEs:

- vscode: make sure `GOPATH` is setup correctly and run `make lint` this will download `golangci-lint` if it was not already installed on your machine. Then add the following to your `settings.json`:
```json
"go.lintTool": "golangci-lint",
"go.lintFlags": [
    "--fast"
],
```

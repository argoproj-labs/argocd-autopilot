# Security Policy for Argo-CD Autopilot

## Preface

Argo-CD Autopilot is a tool that helps users to get an opinionated gitops
repository and bootstrapped Argo-CD installation. To achieve its goals
Argo-CD Autopilot requires access to the Kubernetes cluster you want to
install Argo-CD on and optionally to other Kubernetes clusters you want
to connect to the Argo-CD instance as target clusters for deployments.

Because Argo-CD Autopilot is a gitops tool it also requires access to
your git repositories. Currently it requires pull and push access to
your gitops repo (permission to create repositories is also required
if you want to also create the repository as part of the bootstrapping
process). Though, there are [plans](https://github.com/argoproj-labs/argocd-autopilot/issues/51) 
to have an optional <i>local</i> mode of operation where the user can
tell Argo-CD Autopilot to make changes to a local copy of the repo, 
making the git repository access completely optional.

## Security Scans

We use the following static code analysis tools:

* golangci-lint and tslint for compile time linting
* snyk.io - for image scanning

These are run on each pull request and before each release.

Additionally, Dependabot is configured to scan and report new security 
vulnerabilities in our dependancy tree on a daily basis.

## Reporting a Vulnerability

If you find a security related bug in Argo-CD Autopilot, we kindly ask you 
for responsible disclosure and for giving us appropriate time to react, 
analyze and develop a fix to mitigate the found security vulnerability.

Please report vulnerabilities by e-mail to the following address: 

* argocd-autopilot@codefresh.io

All vulnerabilities and associated information will be treated with full confidentiality. 

## Public Disclosure

Security vulnerabilities will be disclosed via release notes and using the
[GitHub Security Advisories](https://github.com/argoproj-labs/argocd-autopilot/security/advisories)
feature to keep our community well informed, and will credit you for your findings (unless you prefer to stay anonymous, of course).

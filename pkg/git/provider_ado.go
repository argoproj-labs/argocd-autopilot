package git

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	ado "github.com/microsoft/azure-devops-go-api/azuredevops/git"
)

//go:generate mockgen -destination=./ado/mocks/ado.go -package=mocks -source=./provider_ado.go AdoClient,AdoUrl

type (
	AdoClient interface {
		CreateRepository(context.Context, ado.CreateRepositoryArgs) (*ado.GitRepository, error)
	}

	AdoUrl interface {
		GetProjectName() string
	}

	adoGit struct {
		adoClient AdoClient
		opts      *ProviderOptions
		adoUrl    AdoUrl
	}

	adoGitUrl struct {
		loginUrl     string
		subscription string
		projectName  string
	}
)

const Azure = "azure"
const AzureHostName = "dev.azure"
const timeoutTime = 10 * time.Second

func newAdo(opts *ProviderOptions) (Provider, error) {
	adoUrl, err := parseAdoUrl(opts.Host)
	if err != nil {
		return nil, err
	}
	connection := azuredevops.NewPatConnection(adoUrl.loginUrl, opts.Auth.Password)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutTime)
	defer cancel()
	// FYI: ado also has a "core" client that can be used to update project, teams, and other ADO constructs
	gitClient, err := ado.NewClient(ctx, connection)
	if err != nil {
		return nil, err
	}

	return &adoGit{
		adoClient: gitClient,
		opts:      opts,
		adoUrl:    adoUrl,
	}, nil
}

func (g *adoGit) CreateRepository(ctx context.Context, orgRepo string) (string, error) {
	if orgRepo == "" {
		return "", fmt.Errorf("name needs to be provided to create an azure devops repository. "+
			"name: '%s'", orgRepo)
	}

	project := g.adoUrl.GetProjectName()
	createRepositoryArgs := ado.CreateRepositoryArgs{
		GitRepositoryToCreate: &ado.GitRepositoryCreateOptions{
			Name: &orgRepo,
		},
		Project: &project,
	}
	repository, err := g.adoClient.CreateRepository(ctx, createRepositoryArgs)
	if err != nil {
		return "", err
	}

	return *repository.RemoteUrl, nil
}

func (g *adoGit) GetAuthor(ctx context.Context) (username, email string, err error) {
	// empty implementation - will fall back on getting values from global gitconfig file
	return
}

func (a *adoGitUrl) GetProjectName() string {
	return a.projectName
}

// getLoginUrl parses a URL to retrieve the subscription and project name
func parseAdoUrl(host string) (*adoGitUrl, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	var sub, project string
	path := strings.Split(u.Path, "/")
	if len(path) < 5 {
		return nil, fmt.Errorf("unable to parse Azure DevOps url")
	} else {
		// 1 since the path starts with a slash
		sub = path[1]
		project = path[2]
	}
	loginUrl := fmt.Sprintf("%s://%s/%s", u.Scheme, u.Host, sub)
	return &adoGitUrl{
		loginUrl:     loginUrl,
		subscription: sub,
		projectName:  project,
	}, nil
}

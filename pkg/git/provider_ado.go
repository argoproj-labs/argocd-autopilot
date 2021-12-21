package git

import (
	"context"
	"fmt"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	ado "github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"time"
)

//go:generate mockery --name AdoClient --output ado/mocks --case snake
type (
	AdoClient interface {
		CreateRepository(context.Context, ado.CreateRepositoryArgs) (*ado.GitRepository, error)
	}
	adoGit struct {
		adoClient AdoClient
	}
)

const timeoutTime = 10 * time.Second

func newAdo(opts *ProviderOptions) (Provider, error) {
	connection := azuredevops.NewPatConnection(opts.Host, opts.Auth.Password)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutTime)
	defer cancel()
	// FYI: ado also has a "core" client that can be used to update project, teams, and other ADO constructs
	gitClient, err := ado.NewClient(ctx, connection)
	if err != nil {
		return nil, err
	}

	return &adoGit{
		adoClient: gitClient,
	}, nil
}

func (g *adoGit) CreateRepository(ctx context.Context, opts *CreateRepoOptions) (string, error) {
	if opts.Name == "" || opts.Project == "" {
		return "", fmt.Errorf("name and project need to be provided to create an azure devops repository. "+
			"name: '%s' project '%s'", opts.Name, opts.Project)
	}
	gitRepoToCreate := &ado.GitRepositoryCreateOptions{
		Name: &opts.Name,
	}
	createRepositoryArgs := ado.CreateRepositoryArgs{
		GitRepositoryToCreate: gitRepoToCreate,
		Project:               &opts.Project,
	}
	repository, err := g.adoClient.CreateRepository(ctx, createRepositoryArgs)
	if err != nil {
		return "", err
	}
	return *repository.RemoteUrl, nil
}

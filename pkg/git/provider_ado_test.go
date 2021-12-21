package git

import (
	"context"
	"errors"
	"fmt"
	adoMock "github.com/argoproj-labs/argocd-autopilot/pkg/git/ado/mocks"
	ado "github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func Test_adoGit_CreateRepository(t *testing.T) {
	remoteURL := "https://dev.azure.com/SUB/PROJECT/_git/REPO"
	emptyFunc := func(client *adoMock.AdoClient) {}
	type args struct {
		ctx  context.Context
		opts *CreateRepoOptions
	}
	tests := []struct {
		name       string
		mockClient func(client *adoMock.AdoClient)
		args       args
		want       string
		wantErr    assert.ErrorAssertionFunc
	}{
		{name: "Empty Name", mockClient: emptyFunc, args: args{
			ctx: context.TODO(),
			opts: &CreateRepoOptions{
				Owner:   "rumstead",
				Name:    "",
				Project: "project",
			},
		}, want: "", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return true
		}},
		{name: "Empty Project", mockClient: emptyFunc, args: args{
			ctx: context.TODO(),
			opts: &CreateRepoOptions{
				Owner:   "rumstead",
				Name:    "name",
				Project: "",
			},
		}, want: "", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return true
		}},
		{name: "Failure creating repo", mockClient: func(client *adoMock.AdoClient) {
			client.On("CreateRepository", context.TODO(), mock.AnythingOfType("CreateRepositoryArgs")).Return(nil, errors.New("ah an error"))
		}, args: args{
			ctx: context.TODO(),
			opts: &CreateRepoOptions{
				Owner:   "rumstead",
				Name:    "name",
				Project: "project",
			},
		}, want: "", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return true
		}},
		{name: "Success creating repo", mockClient: func(client *adoMock.AdoClient) {
			client.On("CreateRepository", context.TODO(), mock.AnythingOfType("CreateRepositoryArgs")).Return(&ado.GitRepository{
				Links:            nil,
				DefaultBranch:    nil,
				Id:               nil,
				IsFork:           nil,
				Name:             nil,
				ParentRepository: nil,
				Project:          nil,
				RemoteUrl:        &remoteURL,
				Size:             nil,
				SshUrl:           nil,
				Url:              nil,
				ValidRemoteUrls:  nil,
				WebUrl:           nil,
			}, nil)
		}, args: args{
			ctx: context.TODO(),
			opts: &CreateRepoOptions{
				Owner:   "rumstead",
				Name:    "name",
				Project: "project",
			},
		}, want: "https://dev.azure.com/SUB/PROJECT/_git/REPO", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return false
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &adoMock.AdoClient{}
			tt.mockClient(mockClient)
			g := &adoGit{
				adoClient: mockClient,
			}
			got, err := g.CreateRepository(tt.args.ctx, tt.args.opts)
			if !tt.wantErr(t, err, fmt.Sprintf("CreateRepository(%v, %v)", tt.args.ctx, tt.args.opts)) {
				return
			}
			assert.Equalf(t, tt.want, got, "CreateRepository(%v, %v)", tt.args.ctx, tt.args.opts)
		})
	}
}

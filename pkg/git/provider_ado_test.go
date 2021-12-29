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
	emptyFunc := func(client *adoMock.AdoClient, url *adoMock.AdoUrl) {}
	type args struct {
		ctx  context.Context
		opts *CreateRepoOptions
	}
	tests := []struct {
		name       string
		mockClient func(client *adoMock.AdoClient, url *adoMock.AdoUrl)
		args       args
		want       string
		wantErr    assert.ErrorAssertionFunc
	}{
		{name: "Empty Name", mockClient: emptyFunc, args: args{
			ctx: context.TODO(),
			opts: &CreateRepoOptions{
				Owner: "rumstead",
				Name:  "",
			},
		}, want: "", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return true
		}},
		{name: "Failure creating repo", mockClient: func(client *adoMock.AdoClient, url *adoMock.AdoUrl) {
			client.On("CreateRepository", context.TODO(),
				mock.AnythingOfType("CreateRepositoryArgs")).
				Return(nil, errors.New("ah an error"))
			url.On("GetProjectName").Return("blah")
		}, args: args{
			ctx: context.TODO(),
			opts: &CreateRepoOptions{
				Owner: "rumstead",
				Name:  "name",
			},
		}, want: "", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return true
		}},
		{name: "Success creating repo", mockClient: func(client *adoMock.AdoClient, url *adoMock.AdoUrl) {
			url.On("GetProjectName").Return("PROJECT")
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
				Owner: "rumstead",
				Name:  "name",
			},
		}, want: "https://dev.azure.com/SUB/PROJECT/_git/REPO", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return false
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &adoMock.AdoClient{}
			mockUrl := &adoMock.AdoUrl{}
			tt.mockClient(mockClient, mockUrl)
			g := &adoGit{
				adoClient: mockClient,
				adoUrl:    mockUrl,
			}
			got, err := g.CreateRepository(tt.args.ctx, tt.args.opts)
			if !tt.wantErr(t, err, fmt.Sprintf("CreateRepository(%v, %v)", tt.args.ctx, tt.args.opts)) {
				return
			}
			assert.Equalf(t, tt.want, got, "CreateRepository(%v, %v)", tt.args.ctx, tt.args.opts)
		})
	}
}

func Test_parseAdoUrl(t *testing.T) {
	type args struct {
		host string
	}
	tests := []struct {
		name    string
		args    args
		want    *adoGitUrl
		wantErr assert.ErrorAssertionFunc
	}{
		{name: "Invalid URL", args: args{host: "https://dev.azure.com"}, want: nil, wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return true
		}},
		// url taking from the url_test in the url/net module
		{name: "Parse Error", args: args{host: "http://[fe80::%31]:8080/"}, want: nil, wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return true
		}},
		{name: "Parse URL", args: args{host: "https://dev.azure.com/SUB/PROJECT/_git/REPO "}, want: &adoGitUrl{
			loginUrl:     "https://dev.azure.com/SUB",
			subscription: "SUB",
			projectName:  "PROJECT",
		}, wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return false
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAdoUrl(tt.args.host)
			if !tt.wantErr(t, err, fmt.Sprintf("parseAdoUrl(%v)", tt.args.host)) {
				return
			}
			assert.Equalf(t, tt.want, got, "parseAdoUrl(%v)", tt.args.host)
		})
	}
}

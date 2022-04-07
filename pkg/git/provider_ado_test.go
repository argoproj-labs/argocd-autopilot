package git

import (
	"context"
	"errors"
	"fmt"
	"testing"

	adoMock "github.com/argoproj-labs/argocd-autopilot/pkg/git/ado/mocks"

	"github.com/golang/mock/gomock"
	ado "github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/stretchr/testify/assert"
)

func Test_adoGit_CreateRepository(t *testing.T) {
	remoteURL := "https://dev.azure.com/SUB/PROJECT/_git/REPO"
	emptyFunc := func(client *adoMock.MockAdoClient, url *adoMock.MockAdoUrl) {}
	tests := []struct {
		name       string
		mockClient func(client *adoMock.MockAdoClient, url *adoMock.MockAdoUrl)
		repoName   string
		want       string
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name:       "Empty Name",
			mockClient: emptyFunc,
			repoName:   "",
			want:       "",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		{
			name: "Failure creating repo",
			mockClient: func(client *adoMock.MockAdoClient, url *adoMock.MockAdoUrl) {
				client.EXPECT().CreateRepository(context.TODO(), gomock.AssignableToTypeOf(ado.CreateRepositoryArgs{})).
					Times(1).
					Return(nil, errors.New("ah an error"))
				url.EXPECT().GetProjectName().
					Times(1).
					Return("blah")
			},
			repoName: "name",
			want:     "",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		{
			name: "Success creating repo",
			mockClient: func(client *adoMock.MockAdoClient, url *adoMock.MockAdoUrl) {
				url.EXPECT().GetProjectName().
					Times(1).
					Return("PROJECT")
				client.EXPECT().CreateRepository(context.TODO(), gomock.AssignableToTypeOf(ado.CreateRepositoryArgs{})).
					Times(1).
					Return(&ado.GitRepository{
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
			},
			repoName: "name",
			want:     "https://dev.azure.com/SUB/PROJECT/_git/REPO",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := adoMock.NewMockAdoClient(ctrl)
			mockUrl := adoMock.NewMockAdoUrl(ctrl)
			tt.mockClient(mockClient, mockUrl)
			g := &adoGit{
				adoClient: mockClient,
				adoUrl:    mockUrl,
			}
			got, err := g.CreateRepository(context.Background(), tt.repoName)
			if !tt.wantErr(t, err, fmt.Sprintf("CreateRepository - %s", tt.repoName)) {
				return
			}

			assert.Equalf(t, tt.want, got, "CreateRepository - %s", tt.repoName)
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
		{
			name: "Invalid URL",
			args: args{host: "https://dev.azure.com"},
			want: nil, wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		// url taking from the url_test in the url/net module
		{name: "Parse Error",
			args: args{host: "http://[fe80::%31]:8080/"},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		{
			name: "Parse URL",
			args: args{host: "https://dev.azure.com/SUB/PROJECT/_git/REPO "},
			want: &adoGitUrl{
				loginUrl:     "https://dev.azure.com/SUB",
				subscription: "SUB",
				projectName:  "PROJECT",
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
		},
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

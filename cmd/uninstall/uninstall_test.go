package uninstall

import (
	"testing"

	"github.com/codefresh-io/cf-argo/pkg/git"
	mockGit "github.com/codefresh-io/cf-argo/pkg/git/mocks"
	"github.com/codefresh-io/cf-argo/test/utils"
	"github.com/stretchr/testify/mock"
)

func newMockRepo() *mockGit.Repository {
	mockRepo := new(mockGit.Repository)
	mockRepo.On("Add", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).Return(nil)
	mockRepo.On("Commit", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).Return("hash", nil)
	mockRepo.On("Push", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*git.PushOptions")).Return(nil)
	return mockRepo
}

func Test_persistGitopsRepo(t *testing.T) {
	ctx := utils.MockLoggerContext()
	mockRepo := newMockRepo()
	values.GitopsRepo = mockRepo

	msg, gitToken := "some message", "some token"

	persistGitopsRepo(ctx, &options{
		gitToken: gitToken,
	}, msg)

	mockRepo.AssertCalled(t, "Add", ctx, ".")
	mockRepo.AssertCalled(t, "Commit", ctx, msg)
	mockRepo.AssertCalled(t, "Push", ctx, &git.PushOptions{
		Auth: &git.Auth{
			Password: gitToken,
		},
	})
}

func Test_persistGitopsRepo_dryRun(t *testing.T) {
	ctx := utils.MockLoggerContext()
	mockRepo := newMockRepo()
	values.GitopsRepo = mockRepo
	msg := "some message"

	persistGitopsRepo(ctx, &options{
		dryRun: true,
	}, msg)

	mockRepo.AssertCalled(t, "Add", ctx, ".")
	mockRepo.AssertCalled(t, "Commit", ctx, msg)
	mockRepo.AssertNotCalled(t, "Push")
}

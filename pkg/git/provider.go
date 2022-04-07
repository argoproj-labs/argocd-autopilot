package git

import (
	"context"
	"fmt"
	"sort"
)

//go:generate mockgen -destination=./mocks/provider.go -package=mocks -source=./provider.go Provider

type (
	// Provider represents a git provider
	Provider interface {
		// CreateRepository creates the repository in the remote provider and returns a
		// clone url
		CreateRepository(ctx context.Context, orgRepo string) (string, error)

		// GetAuthor gets the authenticated user's name and email address, for making git commits.
		// Returns empty strings if not implemented
		GetAuthor(ctx context.Context) (username, email string, err error)
	}

	Auth struct {
		Username string
		Password string
	}

	// ProviderOptions for a new git provider
	ProviderOptions struct {
		Type string
		Auth *Auth
		Host string
	}

	CreateRepoOptions struct {
		Owner   string
		Name    string
		Private bool
	}

	GetRepoOptions struct {
		Owner string
		Name  string
	}
)

// Errors
var (
	ErrProviderNotSupported = func(providerType string) error {
		return fmt.Errorf("git provider '%s' not supported", providerType)
	}
	ErrAuthenticationFailed = func(err error) error {
		return fmt.Errorf("authentication failed, make sure credentials are correct: %w", err)
	}
)

var supportedProviders = map[string]func(*ProviderOptions) (Provider, error){
	"github": newGithub,
	"gitea":  newGitea,
	"gitlab": newGitlab,
	Azure:    newAdo,
}

// New creates a new git provider
func newProvider(opts *ProviderOptions) (Provider, error) {
	cons, exists := supportedProviders[opts.Type]
	if !exists {
		return nil, ErrProviderNotSupported(opts.Type)
	}

	return cons(opts)
}

func Providers() []string {
	res := make([]string, 0, len(supportedProviders))
	for p := range supportedProviders {
		res = append(res, p)
	}

	sort.Strings(res) // must sort the providers by name, otherwise the codegen is not deterministic
	return res
}

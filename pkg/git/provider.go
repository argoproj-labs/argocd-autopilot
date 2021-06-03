package git

import (
	"context"
	"errors"
	"fmt"
)

//go:generate mockery --name Provider --filename provider.go

type (
	// Provider represents a git provider
	Provider interface {
		// CreateRepository creates the repository in the remote provider and returns a
		// clone url
		CreateRepository(ctx context.Context, opts *CreateRepoOptions) (string, error)
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
	ErrProviderNotSupported = errors.New("git provider not supported")
	ErrAuthenticationFailed = func(err error) error {
		return fmt.Errorf("authentication failed, make sure credentials are correct: %w", err)
	}
)

var supportedProviders = map[string]func(*ProviderOptions) (Provider, error){
	"github": newGithub,
}

// New creates a new git provider
func NewProvider(opts *ProviderOptions) (Provider, error) {
	cons, exists := supportedProviders[opts.Type]
	if !exists {
		return nil, ErrProviderNotSupported
	}

	return cons(opts)
}

func Providers() []string {
	res := make([]string, 0, len(supportedProviders))
	for p := range supportedProviders {
		res = append(res, p)
	}

	return res
}

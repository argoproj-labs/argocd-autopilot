package git

import (
	"context"
	"errors"
	"fmt"
)

type (
	// Provider represents a git provider
	Provider interface {
		// CreateRepository creates the repository in the remote provider and returns a
		// clone url
		CreateRepository(ctx context.Context, opts *CreateRepoOptions) (string, error)

		GetRepository(ctx context.Context, opts *GetRepoOptions) (string, error)
	}

	Auth struct {
		Username string
		Password string
	}

	// Options for a new git provider
	Options struct {
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
		return fmt.Errorf("authentication failed, make sure credetials are correct: %w", err)
	}
)

// New creates a new git provider
func NewProvider(opts *Options) (Provider, error) {
	switch opts.Type {
	case "github":
		return newGithub(opts)
	default:
		return nil, ErrProviderNotSupported
	}
}

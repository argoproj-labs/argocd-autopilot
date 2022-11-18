package git

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"sort"
)

//go:generate mockgen -destination=./mocks/provider.go -package=mocks -source=./provider.go Provider

type (
	// Provider represents a git provider
	Provider interface {
		// CreateRepository creates the repository in the remote provider and returns a
		// clone url
		CreateRepository(ctx context.Context, orgRepo string) (defaultBranch string, err error)

		// GetDefaultBranch returns the default branch of the repository
		GetDefaultBranch(ctx context.Context, orgRepo string) (string, error)

		// GetAuthor gets the authenticated user's name and email address, for making git commits.
		// Returns empty strings if not implemented
		GetAuthor(ctx context.Context) (username, email string, err error)
	}

	Auth struct {
		Username string
		Password string
		CertFile string
	}

	// ProviderOptions for a new git provider
	ProviderOptions struct {
		Type    string
		Auth    *Auth
		RepoURL string
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

	supportedProviders = map[string]func(*ProviderOptions) (Provider, error){
		"bitbucket":     newBitbucket,
		BitbucketServer: newBitbucketServer,
		"github":        newGithub,
		"gitea":         newGitea,
		"gitlab":        newGitlab,
		Azure:           newAdo,
	}
)

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

func DefaultTransportWithCa(certFile string) (*http.Transport, error) {
	rootCAs, err := getRootCas(certFile)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: rootCAs}
	return transport, nil
}

func getRootCas(certFile string) (*x509.CertPool, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed getting system certificates: %w", err)
	}

	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	if certFile == "" {
		return rootCAs, nil
	}

	certs, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed reading certificate from %s: %w", certFile, err)
	}

	// Append our cert to the system pool
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		return nil, fmt.Errorf("failed adding certificate to rootCAs")
	}

	return rootCAs, nil
}

func (a *Auth) GetCertificate() ([]byte, error) {
	if a.CertFile == "" {
		return nil, nil
	}

	return os.ReadFile(a.CertFile)
}

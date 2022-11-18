package git

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"

	"github.com/argoproj-labs/argocd-autopilot/pkg/util"
)

//go:generate mockgen -destination=./bitbucket-server/mocks/httpClient.go -package=mocks -source=./provider_bitbucket-server.go HttpClient

type (
	HttpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	bitbucketServer struct {
		baseURL *url.URL
		c       HttpClient
		opts    *ProviderOptions
	}

	bbError struct {
		Context       string `json:"context"`
		Message       string `json:"message"`
		ExceptionName string `json:"exceptionName"`
	}

	errorBody struct {
		Errors []bbError `json:"errors"`
	}

	createRepoBody struct {
		Name          string `json:"name"`
		Scm           string `json:"scm"`
		DefaultBranch string `json:"defaultBranch"`
		Public        bool   `json:"public"`
	}

	Link struct {
		Name string `json:"name"`
		Href string `json:"href"`
	}

	Links struct {
		Clone []Link `json:"clone"`
	}

	repoResponse struct {
		Slug          string `json:"slug"`
		Name          string `json:"name"`
		Id            int32  `json:"id"`
		DefaultBranch string `json:"defaultBranch"`
		Public        bool   `json:"public"`
		Links         Links  `json:"links"`
	}

	userResponse struct {
		Slug         string `json:"slug"`
		Name         string `json:"name"`
		DisplayName  string `json:"displayName"`
		EmailAddress string `json:"emailAddress"`
	}
)

const BitbucketServer = "bitbucket-server"

var (
	orgRepoReg = regexp.MustCompile("^scm/(~)?([^/]*)/([^/]*)$")
)

func newBitbucketServer(opts *ProviderOptions) (Provider, error) {
	host, _, _, _, _, _, _ := util.ParseGitUrl(opts.RepoURL)
	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{}
	httpClient.Transport, err = DefaultTransportWithCa(opts.Auth.CertFile)
	if err != nil {
		return nil, err
	}

	g := &bitbucketServer{
		baseURL: baseURL,
		c:       httpClient,
		opts:    opts,
	}

	return g, nil
}

func (bbs *bitbucketServer) CreateRepository(ctx context.Context, orgRepo string) (defaultBranch string, err error) {
	noun, owner, name, err := splitOrgRepo(orgRepo)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/%s/repos", noun, owner)
	repo := &repoResponse{}
	err = bbs.requestRest(ctx, http.MethodPost, path, &createRepoBody{
		Name: name,
		Scm:  "git",
	}, repo)
	if err != nil {
		return "", err
	}

	return repo.DefaultBranch, nil
}

func (bbs *bitbucketServer) GetDefaultBranch(ctx context.Context, orgRepo string) (string, error) {
	noun, owner, name, err := splitOrgRepo(orgRepo)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/%s/repos/%s", noun, owner, name)
	repo := &repoResponse{}
	err = bbs.requestRest(ctx, http.MethodGet, path, nil, repo)
	if err != nil {
		return "", err
	}

	defaultBranch := repo.DefaultBranch
	if defaultBranch == "" {
		// fallback in case server response does not include the value at all
		// in both 6.10 and 8.2 i never actually got it in the response, and HAD to use this fallback
		defaultBranch = "master"
	}

	return defaultBranch, nil
}

func (bbs *bitbucketServer) GetAuthor(ctx context.Context) (username, email string, err error) {
	userSlug, err := bbs.whoAmI(ctx)
	if err != nil {
		err = fmt.Errorf("failed getting current user's slug: %w", err)
		return
	}

	user, err := bbs.getUser(ctx, userSlug)
	if err != nil {
		err = fmt.Errorf("failed getting current user: %w", err)
		return
	}

	username = user.DisplayName
	if username == "" {
		username = user.Name
	}

	email = user.EmailAddress
	if email == "" {
		email = user.Slug
	}

	return
}

func (bbs *bitbucketServer) whoAmI(ctx context.Context) (string, error) {
	data, err := bbs.request(ctx, http.MethodGet, "/plugins/servlet/applinks/whoami", nil)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (bbs *bitbucketServer) getUser(ctx context.Context, userSlug string) (*userResponse, error) {
	path := "users/" + userSlug
	user := &userResponse{}
	err := bbs.requestRest(ctx, http.MethodGet, path, nil, user)
	return user, err
}

func (bbs *bitbucketServer) requestRest(ctx context.Context, method, urlPath string, body interface{}, res interface{}) error {
	restPath := path.Join("rest/api/1.0", urlPath)
	data, err := bbs.request(ctx, method, restPath, body)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, res)
}

func (bbs *bitbucketServer) request(ctx context.Context, method, urlPath string, body interface{}) ([]byte, error) {
	var err error

	urlClone := *bbs.baseURL
	urlClone.Path = path.Join(urlClone.Path, urlPath)
	bodyStr := []byte{}
	if body != nil {
		bodyStr, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	request, err := http.NewRequestWithContext(ctx, method, urlClone.String(), bytes.NewBuffer(bodyStr))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer "+bbs.opts.Auth.Password)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := bbs.c.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read from response body: %w", err)
	}

	statusOK := response.StatusCode >= 200 && response.StatusCode < 300
	if !statusOK {
		error := &errorBody{}
		err = json.Unmarshal(data, error)
		if err != nil {
			return nil, fmt.Errorf("failed unmarshalling error body \"%s\". error: %w", data, err)
		}

		return nil, errors.New(error.Errors[0].Message)
	}

	return data, nil
}

func splitOrgRepo(orgRepo string) (noun, owner, name string, err error) {
	split := orgRepoReg.FindStringSubmatch(orgRepo)
	if len(split) == 0 {
		err = fmt.Errorf("invalid Bitbucket url \"%s\" - must be in the form of \"scm/[~]project-or-username/repo-name\"", orgRepo)
		return
	}

	noun = "projects"
	if split[1] == "~" {
		noun = "users"
	}

	owner = split[2]
	name = split[3]
	return
}

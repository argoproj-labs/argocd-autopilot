package git

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/argoproj-labs/argocd-autopilot/pkg/util"
)

//go:generate mockgen -destination=./bitbucket-server/mocks/httpClient.go -package=mocks -source=./provider_bitbucket-server.go HttpClient

type (
	HttpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	bitbucketServer struct {
		baseUrl *url.URL
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
	orgRepoReg = regexp.MustCompile("^scm/(~)?([^/]*)/([^/.]*)(.git)?$")
)

func newBitbucketServer(opts *ProviderOptions) (Provider, error) {
	host, _, _, _, _, _, _ := util.ParseGitUrl(opts.Host)
	baseUrl, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{}
	g := &bitbucketServer{
		baseUrl: baseUrl,
		c:       httpClient,
		opts:    opts,
	}

	return g, nil
}

func (bbs *bitbucketServer) CreateRepository(ctx context.Context, orgRepo string) (string, error) {
	noun, owner, name, err := splitOrgRepo(orgRepo)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/%s/repos", noun, owner)
	repo := &repoResponse{}
	_, err = bbs.request(ctx, http.MethodPost, path, &createRepoBody{
		Name: name,
		Scm:  "git",
	}, repo)
	if err != nil {
		return "", err
	}

	for _, link := range repo.Links.Clone {
		if link.Name == bbs.baseUrl.Scheme {
			return link.Href, nil

		}
	}

	return "", fmt.Errorf("created repo did not contain a valid %s clone url", bbs.baseUrl.Scheme)
}

func (bbs *bitbucketServer) GetDefaultBranch(ctx context.Context, orgRepo string) (string, error) {
	noun, owner, name, err := splitOrgRepo(orgRepo)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/%s/repos/%s", noun, owner, name)
	repo := &repoResponse{}
	_, err = bbs.request(ctx, http.MethodGet, path, nil, repo)
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
	userSlug, err := bbs.request(ctx, http.MethodGet, "/plugins/servlet/applinks/whoami", nil, nil)
	if err != nil {
		return "", err
	}

	return userSlug, nil
}

func (bbs *bitbucketServer) getUser(ctx context.Context, userSlug string) (*userResponse, error) {
	path := "users/" + userSlug
	user := &userResponse{}
	_, err := bbs.request(ctx, http.MethodGet, path, nil, user)
	return user, err
}

func (bbs *bitbucketServer) request(ctx context.Context, method, urlPath string, body interface{}, res interface{}) (string, error) {
	var err error

	urlClone := *bbs.baseUrl
	restApiPath :=  "rest/api/1.0"
	if strings.HasPrefix(urlPath, "/") {
		// if the urlPath is absolute - do not add "rest/api/1.0" before it
		restApiPath = ""
	}

	urlClone.Path = path.Join(urlClone.Path, restApiPath, urlPath)
	bodyStr := []byte{}
	if body != nil {
		bodyStr, err = json.Marshal(body)
		if err != nil {
			return "", err
		}
	}

	request, err := http.NewRequestWithContext(ctx, method, urlClone.String(), bytes.NewBuffer(bodyStr))
	if err != nil {
		return "", err
	}

	request.Header.Set("Authorization", "Bearer "+bbs.opts.Auth.Password)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := bbs.c.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read from response body: %w", err)
	}

	statusOK := response.StatusCode >= 200 && response.StatusCode < 300
	if !statusOK {
		error := &errorBody{}
		err = json.Unmarshal(data, error)
		if err != nil {
			return "", fmt.Errorf("failed unmarshalling error body \"%s\". error: %w", data, err)
		}

		return "", errors.New(error.Errors[0].Message)
	}

	if res != nil {
		err = json.Unmarshal(data, res)
		if err != nil {
			return "", fmt.Errorf("failed unmarshalling body \"%s\". error: %w", data, err)
		}
	}

	return string(data), nil
}

func splitOrgRepo(orgRepo string) (noun, owner, name string, err error) {
	split := orgRepoReg.FindStringSubmatch(orgRepo)
	if len(split) == 0 {
		err = fmt.Errorf("invalid Bitbucket url \"%s\" - must be in the form of \"scm/[~]project-or-username/repo-name[.git]\"", orgRepo)
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

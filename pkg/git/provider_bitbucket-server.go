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
	"regexp"

	"github.com/argoproj-labs/argocd-autopilot/pkg/util"
)

//go:generate mockgen -destination=./bitbucket-server/mocks/httpClient.go -package=mocks -source=./provider_bitbucket-server.go HttpClient

type (
	HttpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	bitbucketServer struct {
		baseURl string
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
	httpClient := &http.Client{}
	g := &bitbucketServer{
		baseURl: host + "rest/api/1.0",
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

	path := fmt.Sprintf("/%s/%s/repos", noun, owner)
	repo := &repoResponse{}
	err = bbs.request(ctx, "POST", path, &createRepoBody{
		Name: name,
		Scm:  "git",
	}, repo)
	if err != nil {
		return "", err
	}

	host, _ := url.Parse(bbs.baseURl)
	for _, link := range repo.Links.Clone {
		if link.Name == host.Scheme {
			return link.Href, nil

		}
	}

	return "", fmt.Errorf("created repo did not contain a valid %s clone url", host.Scheme)
}

func (bbs *bitbucketServer) GetDefaultBranch(ctx context.Context, orgRepo string) (string, error) {
	noun, owner, name, err := splitOrgRepo(orgRepo)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("/%s/%s/repos/%s", noun, owner, name)
	repo := &repoResponse{}
	err = bbs.request(ctx, "GET", path, nil, repo)
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
	user, err := bbs.getUser(ctx, bbs.opts.Auth.Username)
	if err != nil {
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

func (bbs *bitbucketServer) getUser(ctx context.Context, userSlug string) (*userResponse, error) {
	path := "/users/" + userSlug
	user := &userResponse{}
	err := bbs.request(ctx, "GET", path, nil, user)
	return user, err
}

func (bbs *bitbucketServer) request(ctx context.Context, method, path string, body interface{}, res interface{}) error {
	var err error
	finalUrl := bbs.baseURl + path
	bodyStr := []byte{}
	if body != nil {
		bodyStr, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	request, err := http.NewRequestWithContext(ctx, method, finalUrl, bytes.NewBuffer(bodyStr))
	if err != nil {
		return err
	}

	request.SetBasicAuth(bbs.opts.Auth.Username, bbs.opts.Auth.Password)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := bbs.c.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read from response body: %w", err)
	}

	statusOK := response.StatusCode >= 200 && response.StatusCode < 300
	if !statusOK {
		error := &errorBody{}
		err = json.Unmarshal(data, error)
		if err != nil {
			return fmt.Errorf("failed unmarshalling error body \"%s\". error: %w", data, err)
		}

		return errors.New(error.Errors[0].Message)
	}

	return json.Unmarshal(data, res)
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

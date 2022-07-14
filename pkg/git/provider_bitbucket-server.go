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

type (
	bitbucketServer struct {
		baseURl string
		c       *http.Client
		opts    *ProviderOptions
	}

	createRepoBody struct {
		Name          string `json:"name"`
		Scm           string `json:"scm"`
		DefaultBranch string `json:"defaultBranch"`
		Public        bool   `json:"public"`
	}

	repoResponse struct {
		Slug          string `json:"slug"`
		Name          string `json:"name"`
		Id            int32  `json:"id"`
		DefaultBranch string `json:"defaultBranch"`
		Public        bool   `json:"public"`
		Links         struct {
			Clone []struct {
				Name string `json:"name"`
				Href string `json:"href"`
			} `json:"clone"`
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
	}

	userResponse struct {
		Slug         string `json:"slug"`
		Name         string `json:"name"`
		DisplayName  string `json:"displayName"`
		EmailAddress string `json:"emailAddress"`
	}
)

const BitbucketServer = "bitbucket-server"

var orgRepoReg = regexp.MustCompile("scm/(~)?([^/]*)/([^/.]*)(.git)?")

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
	noun, owner, name := splitOrgRepo(orgRepo)
	path := fmt.Sprintf("/%s/%s/repos", noun, owner)
	repo := &repoResponse{}
	err := bbs.request(ctx, "POST", path, &createRepoBody{
		Name: name,
		Scm:  "git",
	}, repo)
	if err != nil {
		return "", err
	}

	httpsCloneLink := bbs.opts.Host
	host, _ := url.Parse(bbs.opts.Host)
	for _, link := range repo.Links.Clone {
		if link.Name == host.Scheme {
			httpsCloneLink = link.Href
		}
	}

	return httpsCloneLink, nil
}

func (bbs *bitbucketServer) GetDefaultBranch(ctx context.Context, orgRepo string) (string, error) {
	noun, owner, name := splitOrgRepo(orgRepo)
	path := fmt.Sprintf("/%s/%s/repos/%s", noun, owner, name)
	repo := &repoResponse{}
	err := bbs.request(ctx, "GET", path, nil, repo)
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

	statusOK := response.StatusCode >= 200 && response.StatusCode < 300
	if !statusOK {
		return errors.New(response.Status)
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read from response body: %w", err)
	}

	return json.Unmarshal(data, res)
}

func splitOrgRepo(orgRepo string) (noun, owner, name string) {
	split := orgRepoReg.FindStringSubmatch(orgRepo)
	noun = "projects"
	if split[1] == "~" {
		noun = "users"
	}

	owner = split[2]
	name = split[3]
	return
}

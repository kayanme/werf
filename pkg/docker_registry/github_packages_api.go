package docker_registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/werf/logboek"
)

type gitHubApi struct{}

func newGitHubApi() gitHubApi {
	return gitHubApi{}
}

type githubApiUser struct {
	Login             string    `json:"login"`
	Id                int       `json:"id"`
	NodeId            string    `json:"node_id"`
	AvatarUrl         string    `json:"avatar_url"`
	GravatarId        string    `json:"gravatar_id"`
	Url               string    `json:"url"`
	HtmlUrl           string    `json:"html_url"`
	FollowersUrl      string    `json:"followers_url"`
	FollowingUrl      string    `json:"following_url"`
	GistsUrl          string    `json:"gists_url"`
	StarredUrl        string    `json:"starred_url"`
	SubscriptionsUrl  string    `json:"subscriptions_url"`
	OrganizationsUrl  string    `json:"organizations_url"`
	ReposUrl          string    `json:"repos_url"`
	EventsUrl         string    `json:"events_url"`
	ReceivedEventsUrl string    `json:"received_events_url"`
	Type              string    `json:"type"`
	SiteAdmin         bool      `json:"site_admin"`
	Name              string    `json:"name"`
	Company           string    `json:"company"`
	Blog              string    `json:"blog"`
	Location          string    `json:"location"`
	Email             string    `json:"email"`
	Hireable          bool      `json:"hireable"`
	Bio               string    `json:"bio"`
	TwitterUsername   string    `json:"twitter_username"`
	PublicRepos       int       `json:"public_repos"`
	PublicGists       int       `json:"public_gists"`
	Followers         int       `json:"followers"`
	Following         int       `json:"following"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Plan              struct {
		Name          string `json:"name"`
		Space         int    `json:"space"`
		Collaborators int    `json:"collaborators"`
		PrivateRepos  int    `json:"private_repos"`
	} `json:"plan"`
}

func (api *gitHubApi) getUser(ctx context.Context, username, token string) (githubApiUser, *http.Response, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)
	resp, respBody, err := doRequest(ctx, http.MethodGet, url, nil, doRequestOptions{
		Headers: map[string]string{
			"Accept":        "application/vnd.github.v3+json",
			"Authorization": fmt.Sprintf("Bearer %s", token),
		},
		AcceptedCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
	})
	if err != nil {
		return githubApiUser{}, resp, err
	}

	var user githubApiUser
	if err := json.Unmarshal(respBody, &user); err != nil {
		return githubApiUser{}, resp, fmt.Errorf("unexpected body %s", string(respBody))
	}

	return user, nil, nil
}

func (api *gitHubApi) deleteOrgContainerPackage(ctx context.Context, orgName, packageName, token string) (*http.Response, error) {
	url := fmt.Sprintf(
		"https://api.github.com/orgs/%s/packages/container/%s",
		orgName, packageName,
	)
	return api.deleteContainerPackage(ctx, url, token)
}

func (api *gitHubApi) deleteOrgContainerPackageVersion(ctx context.Context, orgName, packageName, packageVersionId, token string) (*http.Response, error) {
	url := fmt.Sprintf(
		"https://api.github.com/orgs/%s/packages/container/%s/versions/%s",
		orgName, packageName, packageVersionId,
	)
	return api.deleteContainerPackage(ctx, url, token)
}

func (api *gitHubApi) getOrgContainerPackageVersionId(ctx context.Context, orgName, packageName, tag, token string) (string, *http.Response, error) {
	url := fmt.Sprintf(
		"https://api.github.com/orgs/%s/packages/container/%s/versions",
		orgName,
		packageName,
	)
	return api.getContainerPackageVersionId(ctx, url, tag, token)
}

func (api *gitHubApi) deleteUserContainerPackage(ctx context.Context, packageName, token string) (*http.Response, error) {
	url := fmt.Sprintf("https://api.github.com/user/packages/container/%s", packageName)
	return api.deleteContainerPackage(ctx, url, token)
}

func (api *gitHubApi) deleteUserContainerPackageVersion(ctx context.Context, packageName, packageVersionId, token string) (*http.Response, error) {
	url := fmt.Sprintf("https://api.github.com/user/packages/container/%s/versions/%s", packageName, packageVersionId)
	return api.deleteContainerPackage(ctx, url, token)
}

func (api *gitHubApi) getUserContainerPackageVersionId(ctx context.Context, packageName, tag, token string) (string, *http.Response, error) {
	url := fmt.Sprintf("https://api.github.com/user/packages/container/%s/versions", packageName)
	return api.getContainerPackageVersionId(ctx, url, tag, token)
}

type githubApiVersion struct {
	Id             int       `json:"id"`
	Name           string    `json:"name"`
	Url            string    `json:"url"`
	PackageHtmlUrl string    `json:"package_html_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	HtmlUrl        string    `json:"html_url"`
	Metadata       struct {
		PackageType string `json:"package_type"`
		Container   struct {
			Tags []string `json:"tags"`
		} `json:"container"`
	} `json:"metadata"`
}

func (api *gitHubApi) getContainerPackageVersionId(ctx context.Context, url, tag, token string) (string, *http.Response, error) {
	for page := 0; true; page++ {
		pageUrl := url + fmt.Sprintf("?page=%d&per_page=100", page)
		resp, respBody, err := doRequest(ctx, http.MethodGet, pageUrl, nil, doRequestOptions{
			Headers: map[string]string{
				"Accept":        "application/vnd.github.v3+json",
				"Authorization": fmt.Sprintf("Bearer %s", token),
			},
			AcceptedCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		})
		if err != nil {
			return "", resp, err
		}

		var pageVersionList []githubApiVersion
		if err := json.Unmarshal(respBody, &pageVersionList); err != nil {
			return "", resp, fmt.Errorf("unexpected body %s", string(respBody))
		}

		if len(pageVersionList) == 0 {
			break
		}

		for _, version := range pageVersionList {
			for _, t := range version.Metadata.Container.Tags {
				if t == tag {
					return fmt.Sprint(version.Id), nil, nil
				}
			}
		}
	}

	return "", nil, fmt.Errorf("container package version id for tag %q not found", tag)
}

func (api *gitHubApi) deleteContainerPackage(ctx context.Context, url, token string) (*http.Response, error) {
	resp, _, err := doRequest(ctx, http.MethodDelete, url, nil, doRequestOptions{
		Headers: map[string]string{
			"Accept":        "application/vnd.github.v3+json",
			"Authorization": fmt.Sprintf("Bearer %s", token),
		},
		AcceptedCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			secondsString := resp.Header.Get("Retry-After")
			seconds, err := strconv.Atoi(secondsString)
			if err == nil {
				logboek.Context(ctx).Warn().LogF(
					"WARNING: Secondary rate limit error occurred. Waiting for %d before retrying package delete request...\n",
					seconds,
				)
				time.Sleep(time.Second * time.Duration(seconds))
				return api.deleteContainerPackage(ctx, url, token)
			}
		}

		return resp, err
	}

	return nil, nil
}

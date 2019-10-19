package github

import (
	"context"
	"github.com/google/go-github/v28/github"
)

type Client interface {
	ListReleases(owner, repo string) ([]*github.RepositoryRelease, error)
}

type fakeClient struct {
	releases map[string][]*github.RepositoryRelease
}

func NewFakeClient() *fakeClient {
	return &fakeClient{
		releases: make(map[string][]*github.RepositoryRelease),
	}
}

func (f *fakeClient) AddRelease(owner, repo, tag, tarURL string) {
	f.releases[owner+repo] = append(f.releases[owner+repo], &github.RepositoryRelease{
		TagName: &tag,
		Assets: []github.ReleaseAsset{
			{
				BrowserDownloadURL: &tarURL,
			},
		},
	})
}

func (f *fakeClient) ListReleases(owner, repo string) ([]*github.RepositoryRelease, error) {
	return f.releases[owner+repo], nil
}

type githubClient struct {
	c *github.Client
}

func NewGithubClient(client *github.Client) *githubClient {
	return &githubClient{c: client}
}

func (g *githubClient) ListReleases(owner, repo string) ([]*github.RepositoryRelease, error) {
	releases, _, err := g.c.Repositories.ListReleases(context.Background(), owner, repo, nil)
	return releases, err
}

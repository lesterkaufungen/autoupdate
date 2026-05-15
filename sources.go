package autoupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
)

// Source defines the interface for different update providers.
type Source interface {
	Check(currentVersion string) (*updateResponse, error)
}

// GenericSource represents a standard update server as defined in the original spec.
type GenericSource struct {
	URL     string
	AppID   string
	Headers map[string]string
	Client  *http.Client
	NodeID  string
}

// NewGenericSource creates a new GenericSource.
func NewGenericSource(url, appID string, headers map[string]string) *GenericSource {
	return &GenericSource{
		URL:     url,
		AppID:   appID,
		Headers: headers,
		Client:  http.DefaultClient,
	}
}

func (s *GenericSource) Check(currentVersion string) (*updateResponse, error) {
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	url := fmt.Sprintf("%s/update", s.URL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-App-ID", s.AppID)
	req.Header.Set("X-OS", runtime.GOOS)
	req.Header.Set("X-Arch", runtime.GOARCH)
	req.Header.Set("X-Version", currentVersion)

	for k, v := range s.Headers {
		req.Header.Set(k, v)
	}

	if s.NodeID != "" {
		req.Header.Set("X-Node-ID", s.NodeID)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var upResp updateResponse
	if err := json.NewDecoder(resp.Body).Decode(&upResp); err != nil {
		return nil, err
	}

	return &upResp, nil
}

func matchAsset(name string) bool {
	name = strings.ToLower(name)
	osMatch := strings.Contains(name, runtime.GOOS)
	if !osMatch && runtime.GOOS == "darwin" {
		osMatch = strings.Contains(name, "macos")
	}

	if !osMatch {
		return false
	}

	arch := runtime.GOARCH
	archMatch := strings.Contains(name, arch)
	if !archMatch {
		switch arch {
		case "amd64":
			archMatch = strings.Contains(name, "x86_64") || strings.Contains(name, "x64")
		case "arm64":
			archMatch = strings.Contains(name, "aarch64")
		}
	}

	return archMatch
}

// GitHubSource fetches updates from GitHub Releases.
type GitHubSource struct {
	Owner  string
	Repo   string
	Token  string
	Client *http.Client
}

// NewGitHubSource creates a new GitHubSource.
func NewGitHubSource(owner, repo, token string) *GitHubSource {
	return &GitHubSource{
		Owner:  owner,
		Repo:   repo,
		Token:  token,
		Client: http.DefaultClient,
	}
}

func (s *GitHubSource) Check(currentVersion string) (*updateResponse, error) {
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", s.Owner, s.Repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.Token != "" {
		req.Header.Set("Authorization", "token "+s.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
		Body string `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	version := strings.TrimPrefix(release.TagName, "v")
	if version == currentVersion {
		return &updateResponse{UpdateAvailable: false}, nil
	}

	// Find asset for current OS/ARCH
	var downloadURL string
	var sha256URL string
	for _, asset := range release.Assets {
		if matchAsset(asset.Name) {
			if strings.HasSuffix(asset.Name, ".sha256") {
				sha256URL = asset.BrowserDownloadURL
			} else {
				downloadURL = asset.BrowserDownloadURL
			}
		}
	}

	if downloadURL == "" {
		return &updateResponse{UpdateAvailable: false}, nil
	}

	sha256Sum := ""
	if sha256URL != "" {
		sResp, err := client.Get(sha256URL)
		if err == nil && sResp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(sResp.Body)
			sha256Sum = strings.Fields(string(body))[0]
			sResp.Body.Close()
		}
	}

	return &updateResponse{
		UpdateAvailable: true,
		Version:         version,
		URL:             downloadURL,
		SHA256:          sha256Sum,
	}, nil
}

// GitLabSource fetches updates from GitLab Releases.
type GitLabSource struct {
	BaseURL string // e.g. https://gitlab.com
	ID      string // Project ID or URL-encoded path
	Token   string
	Client  *http.Client
}

// NewGitLabSource creates a new GitLabSource.
func NewGitLabSource(baseURL, id, token string) *GitLabSource {
	return &GitLabSource{
		BaseURL: baseURL,
		ID:      id,
		Token:   token,
		Client:  http.DefaultClient,
	}
}

func (s *GitLabSource) Check(currentVersion string) (*updateResponse, error) {
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	baseURL := s.BaseURL
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	url := fmt.Sprintf("%s/api/v4/projects/%s/releases", baseURL, s.ID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", s.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab api error: %d", resp.StatusCode)
	}

	var releases []struct {
		TagName string `json:"tag_name"`
		Assets  struct {
			Links []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"links"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return &updateResponse{UpdateAvailable: false}, nil
	}

	latest := releases[0]
	version := strings.TrimPrefix(latest.TagName, "v")
	if version == currentVersion {
		return &updateResponse{UpdateAvailable: false}, nil
	}

	var downloadURL string
	var sha256URL string
	for _, link := range latest.Assets.Links {
		if matchAsset(link.Name) {
			if strings.HasSuffix(link.Name, ".sha256") {
				sha256URL = link.URL
			} else {
				downloadURL = link.URL
			}
		}
	}

	if downloadURL == "" {
		return &updateResponse{UpdateAvailable: false}, nil
	}

	sha256Sum := ""
	if sha256URL != "" {
		sResp, err := client.Get(sha256URL)
		if err == nil && sResp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(sResp.Body)
			sha256Sum = strings.Fields(string(body))[0]
			sResp.Body.Close()
		}
	}

	return &updateResponse{
		UpdateAvailable: true,
		Version:         version,
		URL:             downloadURL,
		SHA256:          sha256Sum,
	}, nil
}

// GiteaSource fetches updates from Gitea Releases.
type GiteaSource struct {
	BaseURL string
	Owner   string
	Repo    string
	Token   string
	Client  *http.Client
}

// NewGiteaSource creates a new GiteaSource.
func NewGiteaSource(baseURL, owner, repo, token string) *GiteaSource {
	return &GiteaSource{
		BaseURL: baseURL,
		Owner:   owner,
		Repo:    repo,
		Token:   token,
		Client:  http.DefaultClient,
	}
}

func (s *GiteaSource) Check(currentVersion string) (*updateResponse, error) {
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/releases/latest", s.BaseURL, s.Owner, s.Repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.Token != "" {
		req.Header.Set("Authorization", "token "+s.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitea api error: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	version := strings.TrimPrefix(release.TagName, "v")
	if version == currentVersion {
		return &updateResponse{UpdateAvailable: false}, nil
	}

	var downloadURL string
	var sha256URL string
	for _, asset := range release.Assets {
		if matchAsset(asset.Name) {
			if strings.HasSuffix(asset.Name, ".sha256") {
				sha256URL = asset.BrowserDownloadURL
			} else {
				downloadURL = asset.BrowserDownloadURL
			}
		}
	}

	if downloadURL == "" {
		return &updateResponse{UpdateAvailable: false}, nil
	}

	sha256Sum := ""
	if sha256URL != "" {
		sResp, err := client.Get(sha256URL)
		if err == nil && sResp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(sResp.Body)
			sha256Sum = strings.Fields(string(body))[0]
			sResp.Body.Close()
		}
	}

	return &updateResponse{
		UpdateAvailable: true,
		Version:         version,
		URL:             downloadURL,
		SHA256:          sha256Sum,
	}, nil
}

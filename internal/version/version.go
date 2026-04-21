package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var CurrentVersion = "v1.4.1"

const (
	RepoURL         = "https://api.github.com/repos/gappylul/deployit/releases/latest"
	OperatorRepoURL = "https://api.github.com/repos/gappylul/webapp-operator/releases"
)

type GithubRelease struct {
	TagName string `json:"tag_name"`
}

func newAuthenticatedRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	token := os.Getenv("PAT_TOKEN")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	return req, nil
}

func CheckForUpdate() string {
	client := http.Client{Timeout: 2 * time.Second}
	req, err := newAuthenticatedRequest("GET", RepoURL)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}

	if isNewer(release.TagName, CurrentVersion) {
		return release.TagName
	}
	return ""
}

func GetLatestOperatorVersion() string {
	client := http.Client{Timeout: 2 * time.Second}
	req, err := newAuthenticatedRequest("GET", OperatorRepoURL)
	if err != nil {
		return "latest"
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("⚠ Network error: %v\n", err)
		return "latest"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("⚠ GitHub API returned %d\n", resp.StatusCode)
		return "latest"
	}

	var releases []GithubRelease
	if err = json.NewDecoder(resp.Body).Decode(&releases); err != nil || len(releases) == 0 {
		return "latest"
	}

	return releases[0].TagName
}

func isNewer(remote, local string) bool {
	remoteParts := strings.Split(strings.TrimPrefix(remote, "v"), ".")
	localParts := strings.Split(strings.TrimPrefix(local, "v"), ".")

	for i := 0; i < len(remoteParts) && i < len(localParts); i++ {
		rNum, _ := strconv.Atoi(remoteParts[i])
		lNum, _ := strconv.Atoi(localParts[i])

		if rNum > lNum {
			return true
		}
		if rNum < lNum {
			return false
		}
	}
	return len(remoteParts) > len(localParts)
}

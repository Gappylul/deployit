package version

import (
	"encoding/json"
	"net/http"
	"time"
)

const CurrentVersion = "v1.0.0"
const RepoURL = "https://api.github.com/repos/gappylul/deployit/releases/latest"

type GithubRelease struct {
	TagName string `json:"tag_name"`
}

func CheckForUpdate() string {
	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(RepoURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}

	if release.TagName != CurrentVersion {
		return release.TagName
	}
	return ""
}

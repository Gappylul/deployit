package version

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const CurrentVersion = "v1.1.0"
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

	if isNewer(release.TagName, CurrentVersion) {
		return release.TagName
	}
	return ""
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

package identity

import (
	"fmt"
	"os"
)

// Resolve determines the identity of the current claimant by checking
// CI environment variables in order, falling back to the hostname.
func Resolve() string {
	checks := []struct {
		env    string
		prefix string
	}{
		{"CLAIMENV_HOLDER", ""},
		{"CI_JOB_ID", "gitlab-job-"},
		{"CI_MERGE_REQUEST_IID", "gitlab-mr-"},
		{"GITHUB_RUN_ID", "github-run-"},
		{"BUILD_ID", "jenkins-"},
	}

	for _, c := range checks {
		if v := os.Getenv(c.env); v != "" {
			if c.prefix == "" {
				return v
			}
			return c.prefix + v
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("host-%s", hostname)
}

//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

type testConfig struct {
	Repo             string
	IssueNumber      string
	PRNumber         string
	DiscussionNumber string
}

type ghResult struct {
	Stdout string
	Stderr string
}

func TestGitHubCommentSurfaces(t *testing.T) {
	if os.Getenv("GH_IMPERSONATE_INTEGRATION") != "1" {
		t.Skip("set GH_IMPERSONATE_INTEGRATION=1 to run GitHub integration tests")
	}

	cfg := loadTestConfig(t)
	bin := buildBinary(t)

	user := runImpersonated(t, bin, "api", "user", "--jq", ".login")
	if strings.TrimSpace(user.Stdout) == "" {
		t.Fatalf("expected current user login from impersonated token")
	}

	t.Run("issue_comment", func(t *testing.T) {
		result := runImpersonated(
			t,
			bin,
			"api",
			"-X",
			"POST",
			fmt.Sprintf("repos/%s/issues/%s/comments", cfg.Repo, cfg.IssueNumber),
			"-f",
			"body="+testBody("issue comment"),
			"--jq",
			"{id, html_url, user:.user.login, app:(.performed_via_github_app | {name,slug})}",
		)

		var comment struct {
			ID      int64  `json:"id"`
			HTMLURL string `json:"html_url"`
			User    string `json:"user"`
			App     struct {
				Name string `json:"name"`
				Slug string `json:"slug"`
			} `json:"app"`
		}
		decodeJSON(t, result.Stdout, &comment)
		if comment.User == "" || comment.App.Slug == "" || comment.HTMLURL == "" {
			t.Fatalf("issue comment missing delegated attribution: %+v", comment)
		}
		t.Logf("issue comment: %s", comment.HTMLURL)
	})

	t.Run("pr_timeline_comment", func(t *testing.T) {
		result := runImpersonated(
			t,
			bin,
			"pr",
			"comment",
			cfg.PRNumber,
			"--repo",
			cfg.Repo,
			"--body",
			testBody("PR timeline comment"),
		)
		url := strings.TrimSpace(result.Stdout)
		if !strings.Contains(url, "/pull/") || !strings.Contains(url, "#issuecomment-") {
			t.Fatalf("unexpected PR comment output: %q", result.Stdout)
		}
		comment := issueComment(t, cfg.Repo, issueCommentIDFromURL(t, url))
		if comment.User == "" || comment.App.Slug == "" || comment.HTMLURL == "" {
			t.Fatalf("PR timeline comment missing delegated attribution: %+v", comment)
		}
		t.Logf("PR timeline comment: %s", url)
	})

	t.Run("pr_inline_review_comment", func(t *testing.T) {
		target := firstPRInlineTarget(t, cfg)
		result := runImpersonated(
			t,
			bin,
			"api",
			"-X",
			"POST",
			fmt.Sprintf("repos/%s/pulls/%s/comments", cfg.Repo, cfg.PRNumber),
			"-f",
			"body="+testBody("PR inline review comment"),
			"-f",
			"commit_id="+target.HeadSHA,
			"-f",
			"path="+target.Path,
			"-F",
			fmt.Sprintf("line=%d", target.Line),
			"-f",
			"side=RIGHT",
			"--jq",
			"{id, html_url, user:.user.login}",
		)
		var comment struct {
			ID      int64  `json:"id"`
			HTMLURL string `json:"html_url"`
			User    string `json:"user"`
		}
		decodeJSON(t, result.Stdout, &comment)
		if comment.ID == 0 || comment.HTMLURL == "" || comment.User == "" {
			t.Fatalf("PR inline review comment missing expected fields: %+v", comment)
		}
		t.Logf("PR inline review comment: %s", comment.HTMLURL)
	})

	t.Run("discussion_top_level_comment", func(t *testing.T) {
		discussionID := discussionNodeID(t, cfg)
		result := runImpersonated(
			t,
			bin,
			"api",
			"graphql",
			"-f",
			"discussionId="+discussionID,
			"-f",
			"body="+testBody("discussion top-level comment"),
			"-f",
			"query=mutation($discussionId:ID!, $body:String!) { addDiscussionComment(input:{discussionId:$discussionId, body:$body}) { comment { id url author { login } body } } }",
			"--jq",
			".data.addDiscussionComment.comment",
		)

		var comment struct {
			ID     string `json:"id"`
			URL    string `json:"url"`
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
		}
		decodeJSON(t, result.Stdout, &comment)
		if comment.ID == "" || comment.URL == "" || comment.Author.Login == "" {
			t.Fatalf("discussion comment missing expected fields: %+v", comment)
		}
		t.Logf("discussion top-level comment: %s", comment.URL)
	})

	t.Run("discussion_nested_reply", func(t *testing.T) {
		discussionID := discussionNodeID(t, cfg)
		parent := createDiscussionComment(t, bin, discussionID, testBody("discussion parent for nested reply"))
		reply := runImpersonated(
			t,
			bin,
			"api",
			"graphql",
			"-f",
			"discussionId="+discussionID,
			"-f",
			"replyToId="+parent.ID,
			"-f",
			"body="+testBody("discussion nested reply"),
			"-f",
			"query=mutation($discussionId:ID!, $replyToId:ID!, $body:String!) { addDiscussionComment(input:{discussionId:$discussionId, replyToId:$replyToId, body:$body}) { comment { id url author { login } body } } }",
			"--jq",
			".data.addDiscussionComment.comment",
		)

		var comment discussionComment
		decodeJSON(t, reply.Stdout, &comment)
		if comment.ID == "" || comment.URL == "" || comment.Author.Login == "" {
			t.Fatalf("discussion nested reply missing expected fields: %+v", comment)
		}
		t.Logf("discussion nested reply: %s", comment.URL)
	})
}

func loadTestConfig(t *testing.T) testConfig {
	t.Helper()
	cfg := testConfig{
		Repo:             os.Getenv("GH_IMPERSONATE_TEST_REPO"),
		IssueNumber:      os.Getenv("GH_IMPERSONATE_TEST_ISSUE"),
		PRNumber:         os.Getenv("GH_IMPERSONATE_TEST_PR"),
		DiscussionNumber: os.Getenv("GH_IMPERSONATE_TEST_DISCUSSION"),
	}
	missing := []string{}
	if cfg.Repo == "" {
		missing = append(missing, "GH_IMPERSONATE_TEST_REPO")
	}
	if cfg.IssueNumber == "" {
		missing = append(missing, "GH_IMPERSONATE_TEST_ISSUE")
	}
	if cfg.PRNumber == "" {
		missing = append(missing, "GH_IMPERSONATE_TEST_PR")
	}
	if cfg.DiscussionNumber == "" {
		missing = append(missing, "GH_IMPERSONATE_TEST_DISCUSSION")
	}
	if len(missing) > 0 {
		t.Fatalf("missing integration test env: %s", strings.Join(missing, ", "))
	}
	return cfg
}

func buildBinary(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate integration test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	bin := filepath.Join(t.TempDir(), "gh-impersonate")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/gh-impersonate")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build gh-impersonate: %v\n%s", err, output)
	}
	return bin
}

func runImpersonated(t *testing.T, bin string, args ...string) ghResult {
	t.Helper()
	fullArgs := append([]string{"exec", "--"}, args...)
	cmd := exec.Command(bin, fullArgs...)
	cmd.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gh-impersonate %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(fullArgs, " "), err, stdout.String(), stderr.String())
	}
	return ghResult{Stdout: stdout.String(), Stderr: stderr.String()}
}

func discussionNodeID(t *testing.T, cfg testConfig) string {
	t.Helper()
	owner, name, ok := strings.Cut(cfg.Repo, "/")
	if !ok {
		t.Fatalf("GH_IMPERSONATE_TEST_REPO must be OWNER/REPO, got %q", cfg.Repo)
	}
	query := `query($owner:String!, $name:String!, $number:Int!) { repository(owner:$owner, name:$name) { discussion(number:$number) { id } } }`
	cmd := exec.Command(
		"gh",
		"api",
		"graphql",
		"-f",
		"owner="+owner,
		"-f",
		"name="+name,
		"-F",
		"number="+cfg.DiscussionNumber,
		"-f",
		"query="+query,
		"--jq",
		".data.repository.discussion.id",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve discussion id: %v\n%s", err, output)
	}
	id := strings.TrimSpace(string(output))
	if id == "" || id == "null" {
		t.Fatalf("discussion %s was not found in %s", cfg.DiscussionNumber, cfg.Repo)
	}
	return id
}

type issueCommentResponse struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
	User    string `json:"user"`
	App     struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"app"`
}

func issueComment(t *testing.T, repo string, id string) issueCommentResponse {
	t.Helper()
	cmd := exec.Command(
		"gh",
		"api",
		fmt.Sprintf("repos/%s/issues/comments/%s", repo, id),
		"--jq",
		"{id, html_url, user:.user.login, app:(.performed_via_github_app | {name,slug})}",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fetch issue comment %s: %v\n%s", id, err, output)
	}
	var comment issueCommentResponse
	decodeJSON(t, string(output), &comment)
	return comment
}

func issueCommentIDFromURL(t *testing.T, url string) string {
	t.Helper()
	_, id, ok := strings.Cut(url, "#issuecomment-")
	if !ok || id == "" {
		t.Fatalf("cannot parse issue comment id from %q", url)
	}
	return id
}

type prInlineTarget struct {
	HeadSHA string
	Path    string
	Line    int
}

func firstPRInlineTarget(t *testing.T, cfg testConfig) prInlineTarget {
	t.Helper()
	var pr struct {
		HeadSHA string `json:"head_sha"`
	}
	runGhJSON(t, &pr, "api", fmt.Sprintf("repos/%s/pulls/%s", cfg.Repo, cfg.PRNumber), "--jq", "{head_sha:.head.sha}")
	if pr.HeadSHA == "" {
		t.Fatalf("PR %s has no head SHA", cfg.PRNumber)
	}

	var files []struct {
		Filename string `json:"filename"`
		Patch    string `json:"patch"`
	}
	runGhJSON(t, &files, "api", fmt.Sprintf("repos/%s/pulls/%s/files", cfg.Repo, cfg.PRNumber))
	for _, file := range files {
		line, ok := firstAddedLine(file.Patch)
		if ok {
			return prInlineTarget{HeadSHA: pr.HeadSHA, Path: file.Filename, Line: line}
		}
	}
	t.Fatalf("PR %s has no added line suitable for an inline review comment", cfg.PRNumber)
	return prInlineTarget{}
}

var hunkHeaderPattern = regexp.MustCompile(`@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func firstAddedLine(patch string) (int, bool) {
	currentLine := 0
	for _, line := range strings.Split(patch, "\n") {
		if match := hunkHeaderPattern.FindStringSubmatch(line); match != nil {
			parsed, err := strconv.Atoi(match[1])
			if err != nil {
				return 0, false
			}
			currentLine = parsed
			continue
		}
		if currentLine == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			return currentLine, true
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			continue
		default:
			currentLine++
		}
	}
	return 0, false
}

func runGhJSON(t *testing.T, target any, args ...string) {
	t.Helper()
	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gh %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	decodeJSON(t, string(output), target)
}

type discussionComment struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

func createDiscussionComment(t *testing.T, bin string, discussionID string, body string) discussionComment {
	t.Helper()
	result := runImpersonated(
		t,
		bin,
		"api",
		"graphql",
		"-f",
		"discussionId="+discussionID,
		"-f",
		"body="+body,
		"-f",
		"query=mutation($discussionId:ID!, $body:String!) { addDiscussionComment(input:{discussionId:$discussionId, body:$body}) { comment { id url author { login } body } } }",
		"--jq",
		".data.addDiscussionComment.comment",
	)
	var comment discussionComment
	decodeJSON(t, result.Stdout, &comment)
	if comment.ID == "" {
		t.Fatalf("failed to create parent discussion comment: %s", result.Stdout)
	}
	return comment
}

func decodeJSON(t *testing.T, input string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(input), target); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, input)
	}
}

func testBody(surface string) string {
	return fmt.Sprintf("gh-impersonate integration test: %s at %s", surface, time.Now().UTC().Format(time.RFC3339))
}

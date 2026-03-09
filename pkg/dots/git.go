package dots

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitCloneOpts configures a git clone operation.
type GitCloneOpts struct {
	Branch string
}

// GitClient abstracts Git operations for testability.
type GitClient interface {
	// Clone clones a repository from url into dest.
	Clone(ctx context.Context, url, dest string, opts GitCloneOpts) error
	// Pull fetches and merges the latest changes in the given directory.
	Pull(ctx context.Context, dir string) error
}

// ExecGitClient implements GitClient using exec.CommandContext.
type ExecGitClient struct{}

// NewExecGitClient returns a GitClient backed by the git CLI.
func NewExecGitClient() *ExecGitClient {
	return &ExecGitClient{}
}

func (g *ExecGitClient) Clone(ctx context.Context, url, dest string, opts GitCloneOpts) error {
	args := []string{"clone"}
	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}
	args = append(args, "--", url, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %w: %s", url, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (g *ExecGitClient) Pull(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "pull", "--ff-only")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull in %s: %w: %s", dir, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// StubGitClient is a test double that copies fixture content instead of
// running real git commands. Set CloneFunc/PullFunc to customise behavior.
type StubGitClient struct {
	CloneFunc func(ctx context.Context, url, dest string, opts GitCloneOpts) error
	PullFunc  func(ctx context.Context, dir string) error
}

func (s *StubGitClient) Clone(ctx context.Context, url, dest string, opts GitCloneOpts) error {
	if s.CloneFunc != nil {
		return s.CloneFunc(ctx, url, dest, opts)
	}
	return nil
}

func (s *StubGitClient) Pull(ctx context.Context, dir string) error {
	if s.PullFunc != nil {
		return s.PullFunc(ctx, dir)
	}
	return nil
}

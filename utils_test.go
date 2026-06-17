package main

import "testing"

func TestGitConfigMakeGitUrlUsesGitHubBranchRef(t *testing.T) {
	withMWREL(t, "REL1_43")

	config := GitConfig{
		Type:   "github",
		Repo:   "owner/repo",
		Branch: "feature/test",
	}

	got := config.MakeGitUrl()
	want := "https://github.com/owner/repo/archive/refs/heads/feature/test.tar.gz"
	if got != want {
		t.Fatalf("MakeGitUrl() = %q, want %q", got, want)
	}
}

func TestGitConfigMakeGitUrlUsesGitHubTagRef(t *testing.T) {
	withMWREL(t, "REL1_43")

	config := GitConfig{
		Type: "github",
		Repo: "owner/repo",
		Tag:  "v1.2.3",
	}

	got := config.MakeGitUrl()
	want := "https://github.com/owner/repo/archive/refs/tags/v1.2.3.tar.gz"
	if got != want {
		t.Fatalf("MakeGitUrl() = %q, want %q", got, want)
	}
}

func TestGitConfigMakeGitUrlDefaultsToMWRELBranch(t *testing.T) {
	withMWREL(t, "REL1_43")

	config := GitConfig{
		Type: "github",
		Repo: "owner/repo",
	}

	got := config.MakeGitUrl()
	want := "https://github.com/owner/repo/archive/refs/heads/REL1_43.tar.gz"
	if got != want {
		t.Fatalf("MakeGitUrl() = %q, want %q", got, want)
	}
}

func TestGitConfigMakeGitUrlUsesGitLabBranchRef(t *testing.T) {
	withMWREL(t, "REL1_43")

	config := GitConfig{
		Type:   "gitlab",
		Repo:   "group/subgroup/project",
		Branch: "release/1.0",
	}

	got := config.MakeGitUrl()
	want := "https://gitlab.com/api/v4/projects/group%2Fsubgroup%2Fproject/repository/archive.tar.gz?sha=refs%2Fheads%2Frelease%2F1.0"
	if got != want {
		t.Fatalf("MakeGitUrl() = %q, want %q", got, want)
	}
}

func TestGitConfigMakeGitUrlUsesGitLabTagRef(t *testing.T) {
	withMWREL(t, "REL1_43")

	config := GitConfig{
		Type: "gitlab",
		Repo: "group/project",
		Tag:  "v1.2.3",
	}

	got := config.MakeGitUrl()
	want := "https://gitlab.com/api/v4/projects/group%2Fproject/repository/archive.tar.gz?sha=refs%2Ftags%2Fv1.2.3"
	if got != want {
		t.Fatalf("MakeGitUrl() = %q, want %q", got, want)
	}
}

func withMWREL(t *testing.T, value string) {
	t.Helper()

	original := MWREL
	MWREL = value
	t.Cleanup(func() {
		MWREL = original
	})
}

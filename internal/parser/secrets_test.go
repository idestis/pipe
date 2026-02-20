package parser

import (
	"strings"
	"testing"

	"github.com/getpipe-dev/pipe/internal/model"
)

func TestDetectSecrets_AWSKey(t *testing.T) {
	s := model.Step{ID: "aws", Run: model.RunField{Single: "export AWS_KEY=AKIAIOSFODNN7EXAMPLE"}}
	findings := detectSecrets(s)
	if len(findings) == 0 {
		t.Fatal("expected finding for AWS access key")
	}
}

func TestDetectSecrets_SecretAssignment(t *testing.T) {
	s := model.Step{ID: "tok", Run: model.RunField{Single: `api_key="sk_live_abc123def456"`}}
	findings := detectSecrets(s)
	if len(findings) == 0 {
		t.Fatal("expected finding for secret assignment")
	}
}

func TestDetectSecrets_URLWithCredentials(t *testing.T) {
	s := model.Step{ID: "url", Run: model.RunField{Single: "curl https://admin:s3cret@example.com/api"}}
	findings := detectSecrets(s)
	if len(findings) == 0 {
		t.Fatal("expected finding for URL with credentials")
	}
}

func TestDetectSecrets_PrivateKeyHeader(t *testing.T) {
	s := model.Step{ID: "pk", Run: model.RunField{Single: `echo "-----BEGIN RSA PRIVATE KEY-----"`}}
	findings := detectSecrets(s)
	if len(findings) == 0 {
		t.Fatal("expected finding for private key header")
	}
}

func TestDetectSecrets_GitHubToken(t *testing.T) {
	s := model.Step{ID: "gh", Run: model.RunField{Single: "export GH_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef1234"}}
	findings := detectSecrets(s)
	if len(findings) == 0 {
		t.Fatal("expected finding for GitHub token")
	}
}

func TestDetectSecrets_GitLabToken(t *testing.T) {
	s := model.Step{ID: "gl", Run: model.RunField{Single: "export GL_TOKEN=glpat-xxxxxxxxxxxxxxxxxxxx"}}
	findings := detectSecrets(s)
	if len(findings) == 0 {
		t.Fatal("expected finding for GitLab token")
	}
}

func TestDetectSecrets_BearerToken(t *testing.T) {
	s := model.Step{ID: "bearer", Run: model.RunField{Single: `curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.test"`}}
	findings := detectSecrets(s)
	if len(findings) == 0 {
		t.Fatal("expected finding for Bearer token")
	}
}

func TestDetectSecrets_CleanStep(t *testing.T) {
	s := model.Step{ID: "clean", Run: model.RunField{Single: "echo hello world"}}
	findings := detectSecrets(s)
	if len(findings) != 0 {
		t.Fatalf("expected no findings for clean step, got: %v", findings)
	}
}

func TestDetectSecrets_StringsList(t *testing.T) {
	s := model.Step{ID: "multi", Run: model.RunField{Strings: []string{
		"echo safe",
		"export token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef1234",
	}}}
	findings := detectSecrets(s)
	if len(findings) == 0 {
		t.Fatal("expected finding in strings list")
	}
}

func TestDetectSecrets_SubRuns(t *testing.T) {
	s := model.Step{ID: "subs", Run: model.RunField{SubRuns: []model.SubRun{
		{ID: "safe", Run: "echo ok"},
		{ID: "leaky", Run: "curl https://user:pass@db.example.com"},
	}}}
	findings := detectSecrets(s)
	if len(findings) == 0 {
		t.Fatal("expected finding in sub-runs")
	}
}

func TestSecretWarnings_SensitiveSkipped(t *testing.T) {
	p := &model.Pipeline{
		Steps: []model.Step{
			{ID: "secret", Sensitive: true, Run: model.RunField{Single: "export token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef1234"}},
		},
	}
	warns := SecretWarnings(p)
	if len(warns) != 0 {
		t.Fatalf("expected no warnings when sensitive: true, got: %v", warns)
	}
}

func TestSecretWarnings_NotSensitive(t *testing.T) {
	p := &model.Pipeline{
		Steps: []model.Step{
			{ID: "leaky", Run: model.RunField{Single: "export token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef1234"}},
		},
	}
	warns := SecretWarnings(p)
	if len(warns) == 0 {
		t.Fatal("expected warning for step without sensitive: true")
	}
	if !strings.Contains(warns[0], "sensitive: true") {
		t.Fatalf("expected suggestion about sensitive: true, got: %s", warns[0])
	}
}

package runtime

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildDownloadURL(t *testing.T) {
	currentVer := getAppVersionTag()
	tests := []struct {
		name        string
		inputVer    string
		expectedSub string
	}{
		{
			name:        "Dev version fallback to latest VERSION",
			inputVer:    "v0.0.0-dev",
			expectedSub: "/" + currentVer + "/rag-assets.zip",
		},
		{
			name:        "Empty version fallback to latest VERSION",
			inputVer:    "",
			expectedSub: "/" + currentVer + "/rag-assets.zip",
		},
		{
			name:        "Explicit version",
			inputVer:    "v1.3.0",
			expectedSub: "/v1.3.0/rag-assets.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := BuildDownloadURL(tt.inputVer)
			if !strings.HasSuffix(url, tt.expectedSub) {
				t.Errorf("BuildDownloadURL(%q) = %q; expected to end with %q", tt.inputVer, url, tt.expectedSub)
			}
			if !strings.HasPrefix(url, BaseReleaseURL) {
				t.Errorf("BuildDownloadURL(%q) = %q; expected to start with %q", tt.inputVer, url, BaseReleaseURL)
			}
		})
	}
}

func TestPingRagAssetsDownloadURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network ping test in short mode")
	}
	downloadURL := BuildDownloadURL("")
	t.Logf("Pinging asset download URL: %s", downloadURL)

	client := &http.Client{
		Timeout: 10 * time.Second,
		// Follow redirects (GitHub releases redirect to AWS S3/github-production-release-asset)
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// Make a HEAD request first, fallback to GET with Range if needed
	req, err := http.NewRequest("HEAD", downloadURL, nil)
	if err != nil {
		t.Fatalf("Failed to construct HTTP request: %v", err)
	}
	req.Header.Set("User-Agent", "studyloop-asset-checker")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to ping download URL %s: %v", downloadURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		t.Fatalf("Expected HTTP 200 or 302 for %s, got status: %s", downloadURL, resp.Status)
	}
	t.Logf("Successfully verified asset download URL (Status: %s)", resp.Status)
}

func TestCopyFile_Idempotent(t *testing.T) {
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "src.dll")
	dstFile := filepath.Join(tempDir, "dst.dll")

	content := []byte("mock binary library content for testing")
	if err := os.WriteFile(srcFile, content, 0o644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}

	// 1. First copy should succeed
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("first copyFile failed: %v", err)
	}

	// 2. Second copy to existing identical destination should succeed without error
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("second copyFile failed: %v", err)
	}

	// 3. copyFileWithProgress to existing identical destination should also succeed
	called := false
	err := copyFileWithProgress(srcFile, dstFile, 0, 100, "testing", "detail", func(s string, p int, m, d string) {
		called = true
	})
	if err != nil {
		t.Fatalf("copyFileWithProgress failed on identical file: %v", err)
	}
	if !called {
		t.Errorf("expected progress callback to be called")
	}
}


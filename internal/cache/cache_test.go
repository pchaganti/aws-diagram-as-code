package cache

import (
	"archive/zip"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateFileWithDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	filePath := filepath.Join(tempDir, "testdir", "testfile.txt")
	file, err := createFileWithDirectory(filePath)
	if err != nil {
		t.Errorf("createFileWithDirectory failed: %v", err)
	} else {
		if err := file.Close(); err != nil {
			t.Errorf("Failed to close file: %v", err)
		}
	}

	_, err = os.Stat(filePath)
	if err != nil {
		t.Errorf("File not created: %v", err)
	}
}

func TestLoadEtagCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	etagFilePath := filepath.Join(tempDir, "etag.txt")

	// Test when file doesn't exist
	etag, err := loadEtagCache(etagFilePath)
	if err != nil {
		t.Errorf("loadEtagCache failed when file doesn't exist: %v", err)
	}
	if etag != "" {
		t.Errorf("loadEtagCache returned non-empty string when file doesn't exist")
	}

	// Test when file exists
	err = os.WriteFile(etagFilePath, []byte("test-etag"), 0644)
	if err != nil {
		t.Fatalf("Failed to create etag file: %v", err)
	}
	etag, err = loadEtagCache(etagFilePath)
	if err != nil {
		t.Errorf("loadEtagCache failed when file exists: %v", err)
	}
	if etag != "test-etag" {
		t.Errorf("loadEtagCache returned incorrect etag value: %s", etag)
	}
}

func TestWriteEtagCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	etagFilePath := filepath.Join(tempDir, "etag.txt")
	err = writeEtagCache(etagFilePath, "test-etag")
	if err != nil {
		t.Errorf("writeEtagCache failed: %v", err)
	}

	data, err := os.ReadFile(etagFilePath)
	if err != nil {
		t.Errorf("Failed to read etag file: %v", err)
	}
	if string(data) != "test-etag" {
		t.Errorf("Etag file content incorrect: %s", string(data))
	}
}

func TestFetchFile(t *testing.T) {
	// Test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		etag := r.Header.Get("If-None-Match")
		if etag == "test-etag" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Etag", `"test-etag"`)
		if _, err := fmt.Fprint(w, "test content"); err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Test when no cache exists
	filePath, err := FetchFile(server.URL)
	if err != nil {
		t.Errorf("FetchFile failed when no cache exists: %v", err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read cached file: %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("Cached file content incorrect: %s", string(data))
	}

	// Test when cache exists and etag matches
	filePath, err = FetchFile(server.URL)
	if err != nil {
		t.Errorf("FetchFile failed when cache exists and etag matches: %v", err)
	}
	data, err = os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read cached file: %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("Cached file content incorrect: %s", string(data))
	}

	// Test when cache exists but etag doesn't match
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		etag := r.Header.Get("If-None-Match")
		if etag == "new-etag" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Etag", `"new-etag"`)
		if _, err := fmt.Fprint(w, "new content"); err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	})

	filePath, err = FetchFile(server.URL)
	if err != nil {
		t.Errorf("FetchFile failed when cache exists but etag doesn't match: %v", err)
	}
	data, err = os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read cached file: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("Cached file content incorrect: %s", string(data))
	}
}

func TestExtractZipFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	zipFilePath := filepath.Join(tempDir, "test.zip")
	err = createTestZipFile(zipFilePath)
	if err != nil {
		t.Fatalf("Failed to create test zip file: %v", err)
	}

	extractedPath, err := ExtractZipFile(zipFilePath)
	if err != nil {
		t.Errorf("ExtractZipFile failed: %v", err)
	}

	expectedFilePath := filepath.Join(extractedPath, "test.txt")
	_, err = os.Stat(expectedFilePath)
	if err != nil {
		t.Errorf("Extracted file not found: %v", err)
	}
}

func createTestZipFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	zipWriter := zip.NewWriter(file)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			log.Printf("Failed to close zip writer: %v", err)
		}
	}()

	testFileData := []byte("test content")
	f, err := zipWriter.Create("test.txt")
	if err != nil {
		return err
	}
	_, err = f.Write(testFileData)
	if err != nil {
		return err
	}

	return nil
}

// TestExtractZipFile_ZipSlip verifies the Zip Slip (CWE-22) mitigation: an
// archive whose entry name contains "../" traversal segments must be rejected
// rather than writing outside the extraction directory.
func TestExtractZipFile_ZipSlip(t *testing.T) {
	// Redirect the cache base dir to a temp dir so extraction always runs
	// (ExtractZipFile skips when the content-hash cache dir already exists) and
	// the test neither depends on nor pollutes the real home cache.
	origBase := cacheBaseDir
	t.Cleanup(func() { cacheBaseDir = origBase })
	cacheBaseDir = t.TempDir()

	tempDir, err := os.MkdirTemp("", "zipslip")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	zipFilePath := filepath.Join(tempDir, "evil.zip")
	if err := createMaliciousZipFile(zipFilePath); err != nil {
		t.Fatalf("Failed to create malicious zip file: %v", err)
	}

	_, err = ExtractZipFile(zipFilePath)
	if err == nil {
		t.Fatal("expected ExtractZipFile to reject path-traversal entry, got nil error")
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("expected 'path traversal' error, got: %v", err)
	}

	// The rejected extraction must not leave a partially-populated cache dir
	// behind. If it did, ExtractZipFile's os.Stat short-circuit would treat the
	// leftover dir as a valid cached extraction and return success (nil error)
	// on the next call, silently masking the traversal attempt. A second call
	// must therefore reject again rather than succeed.
	if _, err2 := ExtractZipFile(zipFilePath); err2 == nil {
		t.Fatal("expected second ExtractZipFile call to reject again (partial cache not cleaned up), got nil error")
	} else if !strings.Contains(err2.Error(), "path traversal") {
		t.Errorf("expected 'path traversal' error on retry, got: %v", err2)
	}
}

func createMaliciousZipFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	zipWriter := zip.NewWriter(file)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			log.Printf("Failed to close zip writer: %v", err)
		}
	}()

	// Benign entry followed by a traversal entry, mirroring the reported PoC.
	if f, err := zipWriter.Create("icon.png"); err != nil {
		return err
	} else if _, err := f.Write([]byte("\x89PNG\r\n\x1a\n")); err != nil {
		return err
	}

	traversalName := strings.Repeat("../", 8) + "tmp/PWNED_zipslip_test.txt"
	f, err := zipWriter.Create(traversalName)
	if err != nil {
		return err
	}
	if _, err := f.Write([]byte("ZIPSLIP_ARBITRARY_WRITE_PROOF")); err != nil {
		return err
	}
	return nil
}

func TestIsAllowedRedirectHost(t *testing.T) {
	cases := map[string]bool{
		"github.com":                     true,
		"codeload.github.com":            true, // GitHub archive redirect target
		"objects.githubusercontent.com":  true, // GitHub release asset redirect target
		"raw.githubusercontent.com":      true,
		"githubusercontent.com":          true,
		"d1.awsstatic.com":               true,
		"other.awsstatic.com":            false, // only d1 is pinned; no *.awsstatic.com
		"evil.com":                       false,
		"evilgithub.com":                 false,
		"githubusercontent.com.evil.com": false,
		"attacker.s3.amazonaws.com":      false,
	}
	for host, want := range cases {
		if got := isAllowedRedirectHost(host); got != want {
			t.Errorf("isAllowedRedirectHost(%q) = %v, want %v", host, got, want)
		}
	}
}

// TestFetchFile_BlocksDisallowedRedirect verifies that a redirect to a host
// outside the provider allowlist is rejected rather than silently followed.
func TestFetchFile_BlocksDisallowedRedirect(t *testing.T) {
	origBase := cacheBaseDir
	t.Cleanup(func() { cacheBaseDir = origBase })
	cacheBaseDir = t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example.com/payload.zip", http.StatusFound)
	}))
	defer server.Close()

	_, err := FetchFile(server.URL)
	if err == nil {
		t.Fatal("expected FetchFile to block redirect to disallowed host, got nil error")
	}
	if !strings.Contains(err.Error(), "redirect") {
		t.Errorf("expected a redirect-block error, got: %v", err)
	}
}

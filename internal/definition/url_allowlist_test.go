// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package definition

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsAllowedDefinitionURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"official raw.githubusercontent.com", "https://raw.githubusercontent.com/awslabs/diagram-as-code/main/definitions/definition-for-aws-icons-light.yaml", false},
		{"official github.com", "https://github.com/awslabs/diagram-as-code/releases/download/v1.0.0/defs.yaml", false},
		// d1.awsstatic.com is allowed for ZipFile.Url, but NOT for definition files.
		{"aws icons CDN not allowed for definition files", "https://d1.awsstatic.com/webteam/architecture-icons/defs.yaml", true},
		{"wrong scheme on allowed host", "http://github.com/awslabs/diagram-as-code/defs.yaml", true},
		{"untrusted host", "https://example.com/malicious.yaml", true},
		{"localhost", "http://127.0.0.1:8080/defs.yaml", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsAllowedDefinitionURL(tt.url); (err != nil) != tt.wantErr {
				t.Errorf("IsAllowedDefinitionURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestIsAllowedZipURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"official repo zip", "https://raw.githubusercontent.com/awslabs/diagram-as-code/main/internal/definition/testdata/aws-icons.zip", false},
		// The bundled definition-for-aws-icons file legitimately pulls this.
		{"official AWS icons CDN", "https://d1.awsstatic.com/webteam/architecture-icons/q1-2025/AWS-Architecture-Icon-Decks_02072025.zip", false},
		{"d1 other path not allowed (narrowed)", "https://d1.awsstatic.com/some/other/file.zip", true},
		{"untrusted host", "https://example.com/evil.zip", true},
		{"localhost", "http://127.0.0.1:9192/evil.zip", true},
		{"lookalike host", "https://d1.awsstatic.com.evil.com/evil.zip", true},
		{"wrong scheme on allowed host", "http://d1.awsstatic.com/webteam/architecture-icons/x.zip", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsAllowedZipURL(tt.url); (err != nil) != tt.wantErr {
				t.Errorf("IsAllowedZipURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// TestLoadDefinitions_RejectsUntrustedZipURL verifies that a ZipFile.Url that is
// not on the allowlist is rejected before any network fetch when allowUntrusted
// is false. This closes attack vector #2 (untrusted ZipFile.Url via LocalFile).
func TestLoadDefinitions_RejectsUntrustedZipURL(t *testing.T) {
	dir := t.TempDir()
	defPath := filepath.Join(dir, "defs.yaml")
	content := `Definitions:
  EvilZip:
    Type: Zip
    ZipFile:
      SourceType: url
      Url: http://127.0.0.1:9192/evil.zip
`
	if err := os.WriteFile(defPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp definition file: %v", err)
	}

	ds := &DefinitionStructure{}
	err := ds.LoadDefinitions(defPath, false)
	if err == nil {
		t.Fatal("expected LoadDefinitions to reject untrusted ZipFile.Url, got nil error")
	}
	if !strings.Contains(err.Error(), "ZipFile.Url") {
		t.Errorf("expected a ZipFile.Url allowlist error, got: %v", err)
	}
}

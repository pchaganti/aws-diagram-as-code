// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package definition

import (
	"fmt"
	"strings"
)

// officialDefinitionURLPrefixes are the trusted sources for top-level
// definition (YAML) files. Definition files are maintained in the official repo.
var officialDefinitionURLPrefixes = []string{
	"https://raw.githubusercontent.com/awslabs/diagram-as-code/",
	"https://github.com/awslabs/diagram-as-code/",
}

// officialZipURLPrefixes are the trusted sources for ZipFile.Url icon archives.
// In addition to the official repo, the bundled definition-for-aws-icons file
// pulls the official AWS Architecture Icons deck from the AWS static content CDN,
// so that host must be permitted or the default behavior breaks.
var officialZipURLPrefixes = append(
	[]string{"https://d1.awsstatic.com/"},
	officialDefinitionURLPrefixes...,
)

func matchesAllowedPrefix(url string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

// IsAllowedDefinitionURL restricts top-level definition-file URLs to the
// official repository.
func IsAllowedDefinitionURL(url string) error {
	if matchesAllowedPrefix(url, officialDefinitionURLPrefixes) {
		return nil
	}
	return fmt.Errorf("definition file URL must be from the official repository (https://github.com/awslabs/diagram-as-code/), got: %s. Use --allow-untrusted-definitions to allow untrusted URLs", url)
}

// IsAllowedZipURL restricts ZipFile.Url archive sources to the official
// repository and the AWS Architecture Icons CDN (d1.awsstatic.com).
func IsAllowedZipURL(url string) error {
	if matchesAllowedPrefix(url, officialZipURLPrefixes) {
		return nil
	}
	return fmt.Errorf("ZipFile.Url must be from an official source (https://github.com/awslabs/diagram-as-code/ or https://d1.awsstatic.com/), got: %s. Use --allow-untrusted-definitions to allow untrusted URLs", url)
}

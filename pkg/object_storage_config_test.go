package crontools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadObjectStorageConfigByName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "object-storage.conf")
	err := os.WriteFile(path, []byte(`
[storage]
type = s3
provider = Other
env_auth = false
access_key_id = test-key
secret_access_key = test-secret
region = local
endpoint = http://localhost:7070
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadObjectStorageConfig(path, "storage")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Endpoint != "http://localhost:7070" || cfg.Region != "local" || cfg.AccessKeyID != "test-key" || cfg.SecretAccessKey != "test-secret" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestReadObjectStorageConfigDefaultFirstUsableSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "object-storage.conf")
	err := os.WriteFile(path, []byte(`
[ignored]
type = swift
endpoint = https://swift.example.com

[storage]
access_key_id = first-key
secret_access_key = first-secret
region = first-region
endpoint = https://first.example.com

[backup]
type = s3
access_key_id = second-key
secret_access_key = second-secret
region = second-region
endpoint = https://second.example.com
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadObjectStorageConfig(path, "")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Endpoint != "https://first.example.com" {
		t.Fatalf("unexpected default config: %#v", cfg)
	}
}

func TestReadObjectStorageConfigLegacyFlatFormatStillWorks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "object-storage.conf")
	err := os.WriteFile(path, []byte(`
endpoint = https://objects.example.com
region = us-east-1
access-key-id = old-key
secret-access-key = old-secret
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadObjectStorageConfig(path, "")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AccessKeyID != "old-key" || cfg.SecretAccessKey != "old-secret" {
		t.Fatalf("unexpected legacy config: %#v", cfg)
	}
}

package crontools

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

const DefaultObjectStorageConfigFile = "/etc/cron-tools/object-storage.conf"

type ObjectStorageConfig struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

type namedObjectStorageConfig struct {
	name   string
	config ObjectStorageConfig
}

func ReadObjectStorageConfig(path, name string) (*ObjectStorageConfig, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	configs, err := parseObjectStorageConfigs(fd)
	if err != nil {
		return nil, err
	}

	if name != "" {
		for _, cfg := range configs {
			if cfg.name == name {
				config := cfg.config
				return &config, nil
			}
		}
		return nil, fmt.Errorf("object storage config section %q not found", name)
	}

	if len(configs) == 0 {
		return nil, errors.New("object storage config does not contain any usable s3 section")
	}

	config := configs[0].config
	return &config, nil
}

func parseObjectStorageConfigs(fd *os.File) ([]namedObjectStorageConfig, error) {
	type rawSection struct {
		name      string
		values    map[string]string
		lineNo    int
		hasHeader bool
	}

	normalizeKey := func(key string) string {
		return strings.ReplaceAll(strings.TrimSpace(key), "-", "_")
	}

	finalizeSection := func(section *rawSection) (*namedObjectStorageConfig, error) {
		if section == nil {
			return nil, nil
		}
		if section.hasHeader && section.name == "" {
			return nil, fmt.Errorf("invalid object storage config section on line %d", section.lineNo)
		}

		sectionType := section.values["type"]
		if sectionType == "" {
			sectionType = "s3"
		}
		if sectionType != "s3" {
			return nil, nil
		}

		cfg := namedObjectStorageConfig{
			name: section.name,
			config: ObjectStorageConfig{
				Endpoint:        section.values["endpoint"],
				Region:          section.values["region"],
				AccessKeyID:     section.values["access_key_id"],
				SecretAccessKey: section.values["secret_access_key"],
			},
		}

		switch {
		case cfg.config.Endpoint == "":
			return nil, fmt.Errorf("object storage config section %q missing endpoint", section.name)
		case cfg.config.Region == "":
			return nil, fmt.Errorf("object storage config section %q missing region", section.name)
		case cfg.config.AccessKeyID == "":
			return nil, fmt.Errorf("object storage config section %q missing access_key_id", section.name)
		case cfg.config.SecretAccessKey == "":
			return nil, fmt.Errorf("object storage config section %q missing secret_access_key", section.name)
		}

		return &cfg, nil
	}

	var configs []namedObjectStorageConfig
	var current *rawSection
	scanner := bufio.NewScanner(fd)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, fmt.Errorf("invalid object storage config section on line %d", lineNo)
			}
			cfg, err := finalizeSection(current)
			if err != nil {
				return nil, err
			}
			if cfg != nil {
				configs = append(configs, *cfg)
			}
			current = &rawSection{
				name:      strings.TrimSpace(line[1 : len(line)-1]),
				values:    map[string]string{},
				lineNo:    lineNo,
				hasHeader: true,
			}
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid object storage config line %d", lineNo)
		}

		if current == nil {
			current = &rawSection{values: map[string]string{}, lineNo: lineNo}
		}
		current.values[normalizeKey(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	cfg, err := finalizeSection(current)
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		configs = append(configs, *cfg)
	}

	return configs, nil
}

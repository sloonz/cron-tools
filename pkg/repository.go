package crontools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
)

type CronLogWriter interface {
	io.WriteCloser
	Finalize(parseErr error, footers map[string]string) error
}

type Repository interface {
	Location() string
	ListLoggedJobs(ctx context.Context) ([]string, error)
	ListLogs(ctx context.Context, jobID string) ([]string, error)
	OpenLog(ctx context.Context, jobID, logID string) (io.ReadCloser, error)
	ReadLogMetadata(ctx context.Context, jobID, logID string) (*LogMetadata, error)
	ListJobDescriptions(ctx context.Context) ([]string, error)
	OpenJobDescription(ctx context.Context, name string) (io.ReadCloser, error)
	CreateLogWriter(headers map[string]string, startTime time.Time) (CronLogWriter, error)
}

type LocalRepository interface {
	Repository
	LocalPath() string
}

type RepositoryOptions struct {
	ObjectStorageConfigFile string
}

func OpenRepository(location string, opts RepositoryOptions) (Repository, error) {
	if isObjectStorageLocation(location) {
		return openObjectStorageRepository(location, opts)
	}

	if _, err := os.Stat(location); err != nil {
		return nil, err
	}

	return localRepository{root: location}, nil
}

func isObjectStorageLocation(location string) bool {
	return strings.HasPrefix(location, "s3://") || strings.HasPrefix(location, "s3+")
}

func parseObjectStorageLocation(location string) (configName, trimmed string, err error) {
	if strings.HasPrefix(location, "s3://") {
		return "", strings.TrimPrefix(location, "s3://"), nil
	}
	if !strings.HasPrefix(location, "s3+") {
		return "", "", fmt.Errorf("invalid object storage location %q", location)
	}

	rest := strings.TrimPrefix(location, "s3+")
	configName, trimmed, ok := strings.Cut(rest, "://")
	if !ok || configName == "" || trimmed == "" {
		return "", "", fmt.Errorf("invalid object storage location %q", location)
	}
	return configName, trimmed, nil
}

type entryInfo struct {
	name  string
	isDir bool
}

func listLoggedJobs(listDir func(prefix string) ([]entryInfo, error)) ([]string, error) {
	var walk func(prefix string) ([]string, error)
	walk = func(prefix string) ([]string, error) {
		entries, err := listDir(prefix)
		if err != nil {
			return nil, err
		}

		hasLogs := false
		isLeaf := true
		var jobs []string
		for _, entry := range entries {
			name := entry.name
			if name == "" || name[0] == '.' || name[0] == '_' {
				continue
			}

			if entry.isDir {
				isLeaf = false
				child := name
				if prefix != "" {
					child = path.Join(prefix, name)
				}
				childJobs, err := walk(child)
				if err != nil {
					return nil, err
				}
				jobs = append(jobs, childJobs...)
				continue
			}

			hasLogs = true
		}

		if prefix != "" && (hasLogs || isLeaf) {
			jobs = append(jobs, prefix)
		}
		return jobs, nil
	}

	return walk("")
}

func listJobDescriptions(listDir func(prefix string) ([]entryInfo, error)) ([]string, error) {
	var walk func(prefix string) ([]string, error)
	walk = func(prefix string) ([]string, error) {
		entries, err := listDir(prefix)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrNotExist) {
				return nil, nil
			}
			return nil, err
		}

		var paths []string
		for _, entry := range entries {
			name := entry.name
			if name == "" || name[0] == '.' {
				continue
			}

			child := name
			if prefix != "" {
				child = path.Join(prefix, name)
			}

			if entry.isDir {
				childPaths, err := walk(child)
				if err != nil {
					return nil, err
				}
				paths = append(paths, childPaths...)
				continue
			}

			ext := path.Ext(name)
			if ext == ".json" || ext == ".jsonnet" {
				paths = append(paths, child)
			}
		}
		return paths, nil
	}

	return walk("")
}

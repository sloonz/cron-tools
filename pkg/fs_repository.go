package crontools

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"
)

type localRepository struct {
	root string
}

type localCronLogWriter struct {
	root      string
	headers   map[string]string
	startTime time.Time
	tmpFile   *os.File
}

func (r localRepository) Location() string {
	return r.root
}

func (r localRepository) LocalPath() string {
	return r.root
}

func (r localRepository) ListLoggedJobs(_ context.Context) ([]string, error) {
	return listLoggedJobs(func(prefix string) ([]entryInfo, error) {
		dir := r.root
		if prefix != "" {
			dir = filepath.Join(r.root, filepath.FromSlash(prefix))
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		out := make([]entryInfo, 0, len(entries))
		for _, entry := range entries {
			out = append(out, entryInfo{
				name:  entry.Name(),
				isDir: entry.IsDir(),
			})
		}
		return out, nil
	})
}

func (r localRepository) ListLogs(_ context.Context, jobID string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(r.root, filepath.FromSlash(jobID)))
	if err != nil {
		return nil, err
	}

	logs := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || name == "" || name[0] == '.' {
			continue
		}
		logs = append(logs, name)
	}
	return logs, nil
}

func (r localRepository) OpenLog(_ context.Context, jobID, logID string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(r.root, filepath.FromSlash(jobID), logID))
}

func (r localRepository) ReadLogMetadata(ctx context.Context, jobID, logID string) (*LogMetadata, error) {
	reader, err := r.OpenLog(ctx, jobID, logID)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ParseLogMetadataReader(logID, reader)
}

func (r localRepository) ListJobDescriptions(_ context.Context) ([]string, error) {
	return listJobDescriptions(func(prefix string) ([]entryInfo, error) {
		dir := filepath.Join(r.root, "_jobs")
		if prefix != "" {
			dir = filepath.Join(dir, filepath.FromSlash(prefix))
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		out := make([]entryInfo, 0, len(entries))
		for _, entry := range entries {
			out = append(out, entryInfo{
				name:  entry.Name(),
				isDir: entry.IsDir(),
			})
		}
		return out, nil
	})
}

func (r localRepository) OpenJobDescription(_ context.Context, name string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(r.root, "_jobs", filepath.FromSlash(name)))
}

func (r localRepository) CreateLogWriter(headers map[string]string, startTime time.Time) (CronLogWriter, error) {
	ensureDir := func(p string) error {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return os.MkdirAll(p, 0o755)
		}
		return nil
	}

	if err := ensureDir(r.root); err != nil {
		return nil, err
	}
	if err := ensureDir(filepath.Join(r.root, "_incoming")); err != nil {
		return nil, err
	}
	if err := ensureDir(filepath.Join(r.root, "_err")); err != nil {
		return nil, err
	}
	if err := ensureDir(filepath.Join(r.root, "_invalid")); err != nil {
		return nil, err
	}

	tmpFile, err := os.CreateTemp(filepath.Join(r.root, "_incoming"), "log")
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(tmpFile.Name(), 0o644); err != nil {
		_ = tmpFile.Close()
		return nil, err
	}

	return &localCronLogWriter{
		root:      r.root,
		headers:   headers,
		startTime: startTime,
		tmpFile:   tmpFile,
	}, nil
}

func (w *localCronLogWriter) Write(p []byte) (int, error) {
	return w.tmpFile.Write(p)
}

func (w *localCronLogWriter) Close() error {
	return w.tmpFile.Close()
}

func (w *localCronLogWriter) Finalize(parseErr error, footers map[string]string) error {
	if err := w.tmpFile.Close(); err != nil {
		return err
	}

	targetDirectory := w.headers["Job-ID"]
	if parseErr != nil {
		targetDirectory = "_invalid"
	} else if targetDirectory == "" {
		targetDirectory = "_err"
	}

	targetFile := time.Now().UTC().Format(StoreTimeFormat)
	if !w.startTime.IsZero() {
		targetFile = w.startTime.UTC().Format(StoreTimeFormat)
	}

	targetPath := filepath.Join(w.root, filepath.FromSlash(targetDirectory))
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		if err := os.MkdirAll(targetPath, 0o755); err != nil {
			_ = os.Remove(w.tmpFile.Name())
			return err
		}
	}

	finalFile, err := os.CreateTemp(targetPath, targetFile+".*.gz")
	if err != nil {
		_ = os.Remove(w.tmpFile.Name())
		return err
	}
	finalName := finalFile.Name()
	if err := finalFile.Close(); err != nil {
		_ = os.Remove(w.tmpFile.Name())
		return err
	}
	if err := os.Rename(w.tmpFile.Name(), finalName); err != nil {
		_ = os.Remove(w.tmpFile.Name())
		return err
	}
	return nil
}

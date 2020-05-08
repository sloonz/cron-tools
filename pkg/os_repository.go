package crontools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type objectStorageRepository struct {
	location string
	bucket   string
	prefix   string
	client   *s3.Client
	uploader *transfermanager.Client
}

type objectStorageCronLogWriter struct {
	cancel      context.CancelFunc
	pipeWriter  *io.PipeWriter
	done        chan error
	discardOnly bool
	uploadDone  bool
	uploadErr   error
}

func openObjectStorageRepository(location string, opts RepositoryOptions) (Repository, error) {
	configName, trimmed, err := parseObjectStorageLocation(location)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return nil, fs.ErrInvalid
	}

	prefix := ""
	if len(parts) == 2 {
		prefix = strings.Trim(parts[1], "/")
	}

	if opts.ObjectStorageConfigFile == "" {
		return nil, errors.New("object storage repositories require --object-storage-config-file")
	}

	objectStorageConfig, err := ReadObjectStorageConfig(opts.ObjectStorageConfigFile, configName)
	if err != nil {
		return nil, err
	}

	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(objectStorageConfig.Region),
		config.WithBaseEndpoint(objectStorageConfig.Endpoint),
		config.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			objectStorageConfig.AccessKeyID,
			objectStorageConfig.SecretAccessKey,
			"",
		)),
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	uploader := transfermanager.New(client)

	return objectStorageRepository{
		location: location,
		bucket:   parts[0],
		prefix:   prefix,
		client:   client,
		uploader: uploader,
	}, nil
}

func (r objectStorageRepository) Location() string {
	return r.location
}

func (r objectStorageRepository) ListLoggedJobs(ctx context.Context) ([]string, error) {
	return listLoggedJobs(func(prefix string) ([]entryInfo, error) {
		return r.listEntries(ctx, prefix)
	})
}

func (r objectStorageRepository) ListLogs(ctx context.Context, jobID string) ([]string, error) {
	entries, err := r.listEntries(ctx, jobID)
	if err != nil {
		return nil, err
	}

	logs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.isDir || entry.name == "" || entry.name[0] == '.' {
			continue
		}
		logs = append(logs, entry.name)
	}
	return logs, nil
}

func (r objectStorageRepository) OpenLog(ctx context.Context, jobID, logID string) (io.ReadCloser, error) {
	key := r.objectKey(path.Join(jobID, logID))
	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &r.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (r objectStorageRepository) ReadLogMetadata(ctx context.Context, jobID, logID string) (*LogMetadata, error) {
	reader, err := r.OpenLog(ctx, jobID, logID)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ParseLogMetadataReader(logID, reader)
}

func (r objectStorageRepository) ListJobDescriptions(ctx context.Context) ([]string, error) {
	return listJobDescriptions(func(prefix string) ([]entryInfo, error) {
		base := "_jobs"
		if prefix != "" {
			base = path.Join(base, prefix)
		}
		return r.listEntries(ctx, base)
	})
}

func (r objectStorageRepository) OpenJobDescription(ctx context.Context, name string) (io.ReadCloser, error) {
	key := r.objectKey(path.Join("_jobs", name))
	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &r.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (r objectStorageRepository) CreateLogWriter(headers map[string]string, startTime time.Time) (CronLogWriter, error) {
	jobID := headers["Job-ID"]
	if jobID == "" || startTime.IsZero() {
		return &objectStorageCronLogWriter{discardOnly: true}, nil
	}

	nonce, err := randomHex(8)
	if err != nil {
		return nil, err
	}

	targetFile := startTime.UTC().Format(StoreTimeFormat) + "." + nonce + ".gz"
	key := r.objectKey(path.Join(jobID, targetFile))
	pipeReader, pipeWriter := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := r.uploader.UploadObject(ctx, &transfermanager.UploadObjectInput{
			Bucket: &r.bucket,
			Key:    &key,
			Body:   pipeReader,
		})
		_ = pipeReader.Close()
		done <- err
	}()

	return &objectStorageCronLogWriter{
		cancel:     cancel,
		pipeWriter: pipeWriter,
		done:       done,
	}, nil
}

func (r objectStorageRepository) listEntries(ctx context.Context, prefix string) ([]entryInfo, error) {
	keyPrefix := r.dirKey(prefix)
	input := &s3.ListObjectsV2Input{
		Bucket:    &r.bucket,
		Delimiter: stringPtr("/"),
		Prefix:    &keyPrefix,
	}

	var entries []entryInfo
	pager := s3.NewListObjectsV2Paginator(r.client, input)
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, commonPrefix := range page.CommonPrefixes {
			if commonPrefix.Prefix == nil {
				continue
			}
			entryName := strings.TrimSuffix(strings.TrimPrefix(*commonPrefix.Prefix, keyPrefix), "/")
			if entryName == "" {
				continue
			}
			entries = append(entries, entryInfo{name: entryName, isDir: true})
		}

		for _, object := range page.Contents {
			if object.Key == nil || *object.Key == keyPrefix {
				continue
			}
			entryName := strings.TrimPrefix(*object.Key, keyPrefix)
			if entryName == "" || strings.Contains(entryName, "/") {
				continue
			}
			entries = append(entries, entryInfo{name: entryName})
		}
	}

	return entries, nil
}

func (r objectStorageRepository) objectKey(name string) string {
	cleanName := path.Clean(name)
	if cleanName == "." || cleanName == "" {
		return r.prefix
	}
	if r.prefix == "" {
		return cleanName
	}
	return path.Join(r.prefix, cleanName)
}

func (r objectStorageRepository) dirKey(name string) string {
	if name == "" {
		if r.prefix == "" {
			return ""
		}
		return r.prefix + "/"
	}
	key := r.objectKey(name)
	if key == "" {
		return ""
	}
	if strings.HasSuffix(key, "/") {
		return key
	}
	return key + "/"
}

func stringPtr(v string) *string {
	return &v
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (w *objectStorageCronLogWriter) Write(p []byte) (int, error) {
	if w.discardOnly {
		return len(p), nil
	}

	if err := w.pollUploadResult(); err != nil {
		return 0, err
	}

	n, err := w.pipeWriter.Write(p)
	if err == nil {
		return n, nil
	}
	if !errors.Is(err, io.ErrClosedPipe) {
		return n, err
	}

	if doneErr := w.pollUploadResult(); doneErr != nil {
		return n, doneErr
	}

	return n, err
}

func (w *objectStorageCronLogWriter) Close() error {
	if w.discardOnly {
		return nil
	}
	return w.pipeWriter.Close()
}

func (w *objectStorageCronLogWriter) Finalize(parseErr error, footers map[string]string) error {
	if w.discardOnly {
		return nil
	}
	if parseErr != nil {
		w.cancel()
		_ = w.pipeWriter.CloseWithError(parseErr)
		_ = w.waitUploadResult()
		return nil
	}
	if err := w.pipeWriter.Close(); err != nil {
		return err
	}
	return w.waitUploadResult()
}

func (w *objectStorageCronLogWriter) pollUploadResult() error {
	if w.uploadDone {
		if w.uploadErr == nil {
			return io.ErrClosedPipe
		}
		return w.uploadErr
	}

	select {
	case err := <-w.done:
		w.uploadDone = true
		w.uploadErr = err
		if err == nil {
			return io.ErrClosedPipe
		}
		return err
	default:
		return nil
	}
}

func (w *objectStorageCronLogWriter) waitUploadResult() error {
	if w.uploadDone {
		return w.uploadErr
	}

	w.uploadErr = <-w.done
	w.uploadDone = true
	return w.uploadErr
}

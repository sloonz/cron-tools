package ctstore

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"log"
	"os"
	"time"

	ct "github.com/sloonz/cron-tools/pkg"
	flag "github.com/spf13/pflag"
)

type parserResult struct {
	headers   map[string]string
	startTime time.Time
	footers   map[string]string
	parseErr  error
}

type repositoryDestination struct {
	path     string
	writer   ct.CronLogWriter
	writeErr error
}

type compressedBroadcaster struct {
	buffer       bytes.Buffer
	destinations []*repositoryDestination
	streaming    bool
}

func (b *compressedBroadcaster) Write(p []byte) (int, error) {
	if !b.streaming {
		return b.buffer.Write(p)
	}

	b.writeToDestinations(p)
	return len(p), nil
}

func (b *compressedBroadcaster) StartStreaming(destinations []*repositoryDestination) {
	b.destinations = destinations
	b.streaming = true
	if b.buffer.Len() > 0 {
		b.writeToDestinations(b.buffer.Bytes())
		b.buffer.Reset()
	}
}

func (b *compressedBroadcaster) writeToDestinations(p []byte) {
	for _, destination := range b.destinations {
		if destination == nil || destination.writer == nil || destination.writeErr != nil {
			continue
		}
		if _, err := destination.writer.Write(p); err != nil {
			destination.writeErr = err
			log.Printf("cannot stream log to %s: %v", destination.path, err)
		}
	}
}

func startParser() (*io.PipeWriter, chan parserResult, chan parserResult) {
	readyCh := make(chan parserResult, 1)
	doneCh := make(chan parserResult, 1)

	parserReader, parserWriter := io.Pipe()
	go func() {
		headers := map[string]string{}
		footers := map[string]string{}
		var parseErr error
		var startTime time.Time
		signaled := false

		signalReady := func() {
			if signaled {
				return
			}
			headerCopy := make(map[string]string, len(headers))
			for k, v := range headers {
				headerCopy[k] = v
			}
			readyCh <- parserResult{headers: headerCopy, startTime: startTime}
			signaled = true
		}

		ct.ParseLog(parserReader, ct.LogParser{
			OnFormat: func(timestamp time.Time, format string) {
				if format == ct.FormatLog {
					startTime = timestamp
				}
				if format == ct.FormatFailure {
					signalReady()
				}
			},
			OnMeta: func(_ time.Time, key, val string) {
				if key == "Exit" {
					footers[key] = val
				} else {
					headers[key] = val
					if headers["Job-ID"] != "" && !startTime.IsZero() {
						signalReady()
					}
				}
			},
			OnError: func(err error) {
				parseErr = err
				signalReady()
			},
			OnClose: func() {
				if !signaled {
					signalReady()
				}
				doneCh <- parserResult{
					headers:   headers,
					startTime: startTime,
					footers:   footers,
					parseErr:  parseErr,
				}
			},
		})
	}()

	return parserWriter, readyCh, doneCh
}

func createLogWriters(repoPaths []string, objectStorageConfigFile string, headers map[string]string, startTime time.Time) []*repositoryDestination {
	destinations := make([]*repositoryDestination, 0, len(repoPaths))
	for _, repoPath := range repoPaths {
		repo, err := ct.OpenRepository(repoPath, ct.RepositoryOptions{
			ObjectStorageConfigFile: objectStorageConfigFile,
		})
		if err != nil {
			log.Printf("cannot open repository %s: %v", repoPath, err)
			destinations = append(destinations, &repositoryDestination{
				path:     repoPath,
				writeErr: err,
			})
			continue
		}

		writer, err := repo.CreateLogWriter(headers, startTime)
		if err != nil {
			log.Printf("cannot create log writer for %s: %v", repoPath, err)
			destinations = append(destinations, &repositoryDestination{
				path:     repoPath,
				writeErr: err,
			})
			continue
		}

		destinations = append(destinations, &repositoryDestination{
			path:   repoPath,
			writer: writer,
		})
	}
	return destinations
}

func Main(args []string) {
	flags := flag.NewFlagSet("ct store", flag.ExitOnError)
	objectStorageConfigFilePtr := flags.String("object-storage-config-file", ct.DefaultObjectStorageConfigFile, "path to object storage configuration file for s3:// and s3+NAME:// repositories")
	_ = flags.Parse(args)

	repoPaths := flags.Args()
	if len(repoPaths) == 0 {
		defaultRepo := ct.GetDefaultRepositoryPath()
		if defaultRepo == "" {
			ct.Fatal("cannot find repository")
		}
		repoPaths = []string{defaultRepo}
	}

	parserWriter, readyCh, doneCh := startParser()
	defer parserWriter.Close()

	broadcaster := &compressedBroadcaster{}
	gzipWriter := gzip.NewWriter(broadcaster)

	streaming := false
	var destinations []*repositoryDestination
	var readErr error
	buf := make([]byte, 32*1024)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, writeErr := os.Stdout.Write(chunk); writeErr != nil {
				log.Fatalf("cannot write log to stdout: %s", writeErr.Error())
			}
			if _, writeErr := parserWriter.Write(chunk); writeErr != nil {
				log.Fatalf("cannot feed parser: %s", writeErr.Error())
			}
			if _, writeErr := gzipWriter.Write(chunk); writeErr != nil {
				log.Fatalf("cannot compress log: %s", writeErr.Error())
			}
		}

		if !streaming {
			select {
			case ready := <-readyCh:
				destinations = createLogWriters(repoPaths, *objectStorageConfigFilePtr, ready.headers, ready.startTime)
				broadcaster.StartStreaming(destinations)
				streaming = true
			default:
			}
		}

		if err != nil {
			if !errors.Is(err, io.EOF) {
				readErr = err
			}
			break
		}
	}

	if err := parserWriter.Close(); err != nil {
		log.Fatalf("cannot close parser input: %s", err.Error())
	}
	if err := gzipWriter.Close(); err != nil {
		log.Fatalf("cannot finalize compression: %s", err.Error())
	}

	result := <-doneCh
	if readErr != nil && result.parseErr == nil {
		result.parseErr = readErr
	}
	if !streaming {
		destinations = createLogWriters(repoPaths, *objectStorageConfigFilePtr, result.headers, result.startTime)
		broadcaster.StartStreaming(destinations)
	}

	failed := false
	for _, destination := range destinations {
		if destination == nil {
			continue
		}
		if destination.writer == nil {
			failed = true
			continue
		}

		finalizeErr := result.parseErr
		if finalizeErr == nil && destination.writeErr != nil {
			finalizeErr = destination.writeErr
		}
		err := destination.writer.Finalize(finalizeErr, result.footers)
		if err != nil {
			failed = true
			log.Printf("cannot finalize log in %s: %v", destination.path, err)
			continue
		}
		if destination.writeErr != nil {
			failed = true
		}
	}

	if failed {
		log.Fatalf("some repositories failed")
	}
}

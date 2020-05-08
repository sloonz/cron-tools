package crontools

import (
	"compress/gzip"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

type LogMetadata struct {
	StartTime time.Time         `json:"startTime"`
	EndTime   time.Time         `json:"endTime"`
	LogID     string            `json:"logId"`
	Time      time.Time         `json:"time"`
	Headers   map[string]string `json:"headers"`
	Footers   map[string]string `json:"footers"`
}

func LogIDToTime(logID string) (time.Time, error) {
	return time.Parse(StoreTimeFormat, strings.Split(logID, ".")[0])
}

func ParseLogMetadataReader(logID string, fd io.Reader) (*LogMetadata, error) {
	var r io.Reader
	var err error
	var gzipReader *gzip.Reader
	if filepath.Ext(logID) == ".gz" {
		gzipReader, err = gzip.NewReader(fd)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		r = gzipReader
	} else {
		r = fd
	}

	t, err := LogIDToTime(logID)
	if err != nil {
		return nil, err
	}
	meta := &LogMetadata{
		LogID:   logID,
		Time:    t.UTC(),
		Headers: make(map[string]string),
		Footers: make(map[string]string),
	}

	var parseError error
	ParseLog(r, LogParser{
		OnError: func(err error) {
			parseError = err
		},
		OnFormat: func(timestamp time.Time, format string) {
			if format == FormatLog {
				meta.StartTime = timestamp
			}
		},
		OnMeta: func(timestamp time.Time, key, val string) {
			if key == "Exit" {
				meta.EndTime = timestamp
				meta.Footers[key] = val
			} else {
				meta.Headers[key] = val
			}
		},
	})
	if parseError != nil {
		return nil, parseError
	}

	return meta, nil
}

func ParseLogMetadata(storePath, jobID, logID string) (*LogMetadata, error) {
	fd, err := os.Open(filepath.Join(storePath, filepath.FromSlash(jobID), logID))
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	return ParseLogMetadataReader(logID, fd)
}

func GetDefaultRepositoryPath() string {
	usr, err := user.Current()
	if err == nil {
		usrPath := filepath.Join(usr.HomeDir, ".config", "cron-tools", "repository")
		fi, err := os.Lstat(usrPath)
		if err == nil && (fi.Mode()&os.ModeSymlink) != 0 {
			return usrPath
		}
	}

	p := "/etc/cron-tools/repository"
	fi, err := os.Lstat(p)
	if err == nil && (fi.Mode()&os.ModeSymlink) != 0 {
		return p
	}

	return ""
}

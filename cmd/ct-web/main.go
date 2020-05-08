package ctweb

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	ct "github.com/sloonz/cron-tools/pkg"
	web "github.com/sloonz/cron-tools/web"
	flag "github.com/spf13/pflag"
	bolt "go.etcd.io/bbolt"
)

const logsBucket = "logs"

type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

type LoggedJob struct {
	JobID        string          `json:"jobId"`
	FailedCount  uint            `json:"failedCount"`
	SuccessCount uint            `json:"successCount"`
	LastLog      *ct.LogMetadata `json:"lastLog"`
	LastFail     *ct.LogMetadata `json:"lastFail"`
	LastSuccess  *ct.LogMetadata `json:"lastSuccess"`
}

type RegisteredJob struct {
	Description     ct.JobDescription `json:"description"`
	PreviousRunTime time.Time         `json:"previousRunTime"`
	NextRunTime     time.Time         `json:"nextRunTime"`
}

type SyncResult struct {
	HasSync  bool    `json:"hasSync"`
	Success  bool    `json:"success"`
	Output   *string `json:"output"`
	Error    *string `json:"error"`
	SyncDate string  `json:"syncDate"`
}

type APIError struct {
	Error string `json:"error"`
}

type txManager struct {
	db     *bolt.DB
	bucket []byte
	rb     *bolt.Bucket
	wb     *bolt.Bucket
	rtx    *bolt.Tx
	wtx    *bolt.Tx
}

func openBucket(db *bolt.DB, bucket []byte, fn func(tm *txManager) error) error {
	var err error

	tm := new(txManager)
	tm.db = db
	tm.bucket = bucket
	tm.rtx, err = db.Begin(false)
	if err != nil {
		return err
	}
	tm.rb = tm.rtx.Bucket(bucket)
	defer (func() {
		if tm.rtx != nil {
			_ = tm.rtx.Rollback()
		}
		if tm.wtx != nil {
			_ = tm.wtx.Rollback()
		}
	})()

	err = fn(tm)
	if err == nil && tm.wtx != nil {
		err = tm.wtx.Commit()
	}

	return err
}

func (tm *txManager) Get(key []byte) []byte {
	if tm.wtx != nil {
		return tm.wb.Get(key)
	}
	return tm.rb.Get(key)
}

func (tm *txManager) Put(key, val []byte) error {
	var err error
	if tm.wtx == nil {
		_ = tm.rtx.Rollback()
		tm.rtx = nil
		tm.rb = nil
		tm.wtx, err = tm.db.Begin(true)
		if err != nil {
			return err
		}
		tm.wb = tm.wtx.Bucket(tm.bucket)
	}
	return tm.wb.Put(key, val)
}

func getLogMetadata(tm *txManager, repo ct.Repository, jobID, logID string) (*ct.LogMetadata, error) {
	meta := &ct.LogMetadata{}
	k := []byte(jobID + "/" + logID)
	data := tm.Get(k)
	if data != nil {
		err := json.Unmarshal(append([]byte{}, data...), meta)
		if err == nil {
			return meta, nil
		} else {
			log.Printf("error while getting %s from cache: %v", string(k), err)
		}
	}

	meta, err := repo.ReadLogMetadata(context.Background(), jobID, logID)
	if err != nil {
		return nil, err
	}

	m, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	err = tm.Put(k, m)
	if err != nil {
		log.Printf("error while caching %s: %v", string(k), err)
	}

	return meta, nil
}

func getLoggedJobList(db *bolt.DB, repo ct.Repository) ([]LoggedJob, error) {
	var jl []LoggedJob
	err := openBucket(db, []byte(logsBucket), func(tm *txManager) error {
		jobIDs, err := repo.ListLoggedJobs(context.Background())
		if err != nil {
			return err
		}

		for _, jobID := range jobIDs {
			entry := LoggedJob{JobID: jobID}
			logIDs, err := repo.ListLogs(context.Background(), jobID)
			if err != nil {
				return err
			}

			var lastLog, lastFail, lastSuccess *ct.LogMetadata
			for _, logID := range logIDs {
				meta, err := getLogMetadata(tm, repo, jobID, logID)
				if err != nil {
					log.Printf("cannot parse %s/%s: %v", jobID, logID, err)
					continue
				}
				if meta.Footers["Exit"] == "code 0" {
					entry.SuccessCount++
					if lastSuccess == nil || meta.Time.Sub(lastSuccess.Time) > 0 {
						lastSuccess = meta
					}
				} else {
					entry.FailedCount++
					if lastFail == nil || meta.Time.Sub(lastFail.Time) > 0 {
						lastFail = meta
					}
				}
				if lastLog == nil || meta.Time.Sub(lastLog.Time) > 0 {
					lastLog = meta
				}
			}

			entry.LastLog = lastLog
			entry.LastFail = lastFail
			entry.LastSuccess = lastSuccess
			jl = append(jl, entry)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return jl, nil
}

func getLogList(db *bolt.DB, repo ct.Repository, jobID string) ([]ct.LogMetadata, error) {
	logIDs, err := repo.ListLogs(context.Background(), jobID)
	if err != nil {
		return nil, err
	}

	var ll []ct.LogMetadata
	err = openBucket(db, []byte(logsBucket), func(tm *txManager) error {
		for _, logID := range logIDs {
			meta, err := getLogMetadata(tm, repo, jobID, logID)
			if err != nil {
				log.Printf("cannot parse %s/%s: %v", jobID, logID, err)
				continue
			}
			ll = append(ll, *meta)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return ll, nil
}

func getLog(repo ct.Repository, jobID, logID string) (io.ReadCloser, error) {
	return repo.OpenLog(context.Background(), jobID, logID)
}

func getRegisteredJobList(repo ct.Repository, paths []string) ([]RegisteredJob, error) {
	var dl []ct.JobDescription
	for _, p := range paths {
		err := ct.ReadJobDescriptions(p, &dl)
		if err != nil {
			return nil, err
		}
	}
	jobDescriptions, err := repo.ListJobDescriptions(context.Background())
	if err != nil {
		return nil, err
	}
	for _, name := range jobDescriptions {
		fd, err := repo.OpenJobDescription(context.Background(), name)
		if err != nil {
			return nil, err
		}
		err = ct.ReadJobDescriptionsFile(name, fd, &dl)
		closeErr := fd.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}

	jl := make([]RegisteredJob, 0, len(dl))
	for _, d := range dl {
		loc, err := time.LoadLocation(d.Location)
		if err != nil {
			return nil, err
		}
		now := time.Now().In(loc)

		spec, err := ct.ParseCronTimeSpecification(d.Frequency)
		if err != nil {
			return nil, err
		}

		jl = append(jl, RegisteredJob{
			Description:     d,
			NextRunTime:     spec.NextDate(now).UTC(),
			PreviousRunTime: spec.PreviousDate(now).UTC(),
		})
	}

	return jl, nil
}

func stringPtr(s string) *string {
	return &s
}

func defaultCachePath(repo ct.Repository) string {
	if localRepo, ok := repo.(ct.LocalRepository); ok {
		return filepath.Join(localRepo.LocalPath(), ".cache")
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		cacheRoot = os.TempDir()
	}
	cacheKey := sha256.Sum256([]byte(repo.Location()))
	return filepath.Join(cacheRoot, "cron-tools", "ct-web", hex.EncodeToString(cacheKey[:])+".cache")
}

func writeError(w http.ResponseWriter, err error) bool {
	if err != nil {
		log.Print(err)
		writeResponse(w, http.StatusInternalServerError, APIError{Error: err.Error()})
		return true
	}
	return false
}

func writeResponse(w http.ResponseWriter, status int, res any) {
	data, err := json.Marshal(res)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(data)
	if err != nil {
		log.Print(err)
	}
}

func Main(args []string) {
	flags := flag.NewFlagSet("ct web", flag.ExitOnError)
	listenPtr := flags.StringP("listen", "l", "127.0.0.1:3000", "Listen address")
	repoPathPtr := flags.StringP("repository", "r", ct.GetDefaultRepositoryPath(), "path to logs repository")
	cachePathPtr := flags.String("cache", "", "path to local metadata cache database")
	objectStorageConfigFilePtr := flags.String("object-storage-config-file", ct.DefaultObjectStorageConfigFile, "path to object storage configuration file for s3:// and s3+NAME:// repositories")
	_ = flags.Parse(args)

	repoPath := *repoPathPtr
	repo, err := ct.OpenRepository(repoPath, ct.RepositoryOptions{
		ObjectStorageConfigFile: *objectStorageConfigFilePtr,
	})
	if err != nil {
		log.Fatal("cannot find repository")
	}

	jobDescDirs := flags.Args()

	cachePath := *cachePathPtr
	if cachePath == "" {
		cachePath = defaultCachePath(repo)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		log.Fatal(err)
	}

	db, err := bolt.Open(cachePath, 0o600, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(logsBucket))
		return err
	})
	if err != nil {
		_ = db.Close()
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Print(err)
		}
	}()

	localRepo, hasLocalRepo := repo.(ct.LocalRepository)
	router := chi.NewRouter()
	router.Post("/api/sync", func(w http.ResponseWriter, _ *http.Request) {
		if !hasLocalRepo {
			writeResponse(w, http.StatusOK, SyncResult{
				HasSync:  false,
				Success:  true,
				SyncDate: time.Now().Format(time.RFC3339),
			})
			return
		}

		syncPath := filepath.Join(localRepo.LocalPath(), "_sync")
		_, err := os.Stat(syncPath)
		if err != nil {
			if os.IsNotExist(err) {
				writeResponse(w, http.StatusOK, SyncResult{
					HasSync:  false,
					Success:  true,
					SyncDate: time.Now().Format(time.RFC3339),
				})
			} else {
				writeResponse(w, http.StatusOK, SyncResult{
					HasSync:  true,
					Success:  false,
					Error:    stringPtr(err.Error()),
					SyncDate: time.Now().Format(time.RFC3339),
				})
			}
		} else {
			cmd := exec.Command(syncPath)
			out, err := cmd.CombinedOutput()
			if err != nil {
				writeResponse(w, http.StatusOK, SyncResult{
					HasSync:  true,
					Success:  false,
					Output:   stringPtr(string(out)),
					Error:    stringPtr(err.Error()),
					SyncDate: time.Now().Format(time.RFC3339),
				})
			} else {
				writeResponse(w, http.StatusOK, SyncResult{
					HasSync:  true,
					Success:  true,
					Output:   stringPtr(string(out)),
					SyncDate: time.Now().Format(time.RFC3339),
				})
			}
		}

	})
	router.Get("/api/jobs", func(w http.ResponseWriter, _ *http.Request) {
		jl, err := getRegisteredJobList(repo, jobDescDirs)
		if writeError(w, err) {
			return
		}

		writeResponse(w, http.StatusOK, jl)
	})
	router.Get("/api/logs", func(w http.ResponseWriter, _ *http.Request) {
		jl, err := getLoggedJobList(db, repo)
		if writeError(w, err) {
			return
		}

		writeResponse(w, http.StatusOK, jl)
	})
	router.Get("/api/logs/{job}", func(w http.ResponseWriter, r *http.Request) {
		jobID, err := url.PathUnescape(chi.URLParam(r, "job"))
		if writeError(w, err) {
			return
		}

		ll, err := getLogList(db, repo, jobID)
		if writeError(w, err) {
			return
		}

		writeResponse(w, http.StatusOK, ll)
	})
	router.Get("/api/logs/{job}/{log}", func(w http.ResponseWriter, r *http.Request) {
		jobID, err := url.PathUnescape(chi.URLParam(r, "job"))
		if writeError(w, err) {
			return
		}

		logID, err := url.PathUnescape(chi.URLParam(r, "log"))
		if writeError(w, err) {
			return
		}

		logReader, err := getLog(repo, jobID, logID)
		if writeError(w, err) {
			return
		}
		defer func() {
			if err := logReader.Close(); err != nil {
				log.Print(err)
			}
		}()

		var reader io.Reader = logReader
		var gzipReader *gzip.Reader
		if filepath.Ext(logID) == ".gz" {
			gzipReader, err = gzip.NewReader(logReader)
			if writeError(w, err) {
				return
			}
			defer func() {
				if err := gzipReader.Close(); err != nil {
					log.Print(err)
				}
			}()
			reader = gzipReader
		}

		w.Header().Add("Content-Type", "application/cron-log")
		_, err = io.Copy(w, reader)
		if err != nil {
			log.Print(err)
		}
	})
	router.Get("/api/*", func(w http.ResponseWriter, r *http.Request) {
		writeResponse(w, http.StatusNotFound, APIError{Error: "not found"})
	})

	router.Get("/*", http.FileServer(http.FS(fsFunc(func(name string) (fs.File, error) {
		f, err := web.Assets.Open(path.Join("build", name))
		if os.IsNotExist(err) {
			return web.Assets.Open("build/index.html")
		}

		return f, err
	}))).ServeHTTP)

	network := "tcp"
	addr := *listenPtr
	if strings.HasPrefix(*listenPtr, "unix:") {
		network = "unix"
		addr = strings.TrimPrefix(*listenPtr, "unix:")
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			log.Fatal(err)
		}
	}

	listener, err := net.Listen(network, addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on %s:%s", network, addr)
	err = http.Serve(listener, router)
	if err != nil {
		log.Fatal(err)
	}
}

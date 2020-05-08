package ctdigest

import (
	"context"
	"encoding/json"

	ct "github.com/sloonz/cron-tools/pkg"
	flag "github.com/spf13/pflag"

	"log"
	"os"
	"time"
)

type jobSummary struct {
	Description     ct.JobDescription `json:"description"`
	LastLog         *ct.LogMetadata   `json:"lastLog"`
	PreviousRunTime time.Time         `json:"previousRunTime"`
	NextRunTime     time.Time         `json:"nextRunTime"`
}

func readJobsFromRepository(ctx context.Context, repo ct.Repository, jobs *[]ct.JobDescription) error {
	jobDescriptions, err := repo.ListJobDescriptions(ctx)
	if err != nil {
		return err
	}

	for _, name := range jobDescriptions {
		fd, err := repo.OpenJobDescription(ctx, name)
		if err != nil {
			return err
		}
		err = ct.ReadJobDescriptionsFile(name, fd, jobs)
		closeErr := fd.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}

	return nil
}

func Main(args []string) {
	flags := flag.NewFlagSet("ct digest", flag.ExitOnError)
	repoPathPtr := flags.StringP("repository", "r", ct.GetDefaultRepositoryPath(), "path to logs repository")
	objectStorageConfigFilePtr := flags.String("object-storage-config-file", ct.DefaultObjectStorageConfigFile, "path to object storage configuration file for s3:// and s3+NAME:// repositories")
	_ = flags.Parse(args)

	repoPath := *repoPathPtr
	repo, err := ct.OpenRepository(repoPath, ct.RepositoryOptions{
		ObjectStorageConfigFile: *objectStorageConfigFilePtr,
	})
	if err != nil {
		log.Fatal("cannot find repository")
	}

	var jobs []ct.JobDescription
	if len(flags.Args()) == 0 {
		err := readJobsFromRepository(context.Background(), repo, &jobs)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		for _, repoPath := range flags.Args() {
			jobRepo, err := ct.OpenRepository(repoPath, ct.RepositoryOptions{
				ObjectStorageConfigFile: *objectStorageConfigFilePtr,
			})
			if err != nil {
				log.Fatal(err)
			}
			err = readJobsFromRepository(context.Background(), jobRepo, &jobs)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	summaries := make([]jobSummary, len(jobs))
	for i, job := range jobs {
		logs, err := repo.ListLogs(context.Background(), job.ID)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("warning: cannot read %s: %v", job.ID, err)
			}
			summaries[i] = jobSummary{Description: job}
			continue
		}

		var lastLog *ct.LogMetadata
		var lastLogID string
		var lastLogTime time.Time
		for _, logID := range logs {
			t, err := ct.LogIDToTime(logID)
			if err != nil {
				log.Printf("warning: invalid log name %s: %v", logID, err)
				continue
			}

			if lastLogID == "" || lastLogTime.Sub(t) < 0 {
				lastLogID = logID
				lastLogTime = t
			}
		}

		if lastLogID != "" {
			lastLog, err = repo.ReadLogMetadata(context.Background(), job.ID, lastLogID)
			if err != nil {
				log.Printf("warning: cannot parse %s/%s: %v", job.ID, lastLogID, err)
			}
		}

		summaries[i] = jobSummary{
			Description: job,
			LastLog:     lastLog,
		}

		spec, err := ct.ParseCronTimeSpecification(job.Frequency)
		if err != nil {
			log.Printf("warning: cannot parse cron timedate: %s: %v", job.Frequency, err)
		} else {
			loc, err := time.LoadLocation(job.Location)
			if err != nil {
				log.Printf("warning: cannot parse timezone %s: %v", job.Location, err)
			} else {
				now := time.Now().In(loc)
				summaries[i].NextRunTime = spec.NextDate(now).UTC()
				summaries[i].PreviousRunTime = spec.PreviousDate(now).UTC()
			}
		}
	}

	err = json.NewEncoder(os.Stdout).Encode(summaries)
	if err != nil {
		log.Fatal(err)
	}
}

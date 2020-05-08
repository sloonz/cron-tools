package crontools

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/google/go-jsonnet"
)

type JobDescription struct {
	ID        string         `json:"id"`
	Frequency string         `json:"frequency"`
	User      string         `json:"user"`
	Host      string         `json:"host"`
	Command   []string       `json:"command"`
	Location  string         `json:"tz"`
	Extra     map[string]any `json:"extra"`
}

func ReadJobDescriptionsFile(filename string, r io.Reader, jobs *[]JobDescription) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	ast, err := jsonnet.SnippetToAST(filename, string(data))
	if err != nil {
		return err
	}

	vm := jsonnet.MakeVM()
	jsonData, err := vm.Evaluate(ast)
	if err != nil {
		return err
	}

	var importedJobs []JobDescription
	err = json.Unmarshal([]byte(jsonData), &importedJobs)
	if err != nil {
		return err
	}

	*jobs = append(*jobs, importedJobs...)

	return nil
}

func ReadJobDescriptions(p string, jobs *[]JobDescription) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		files, err := os.ReadDir(p)
		if err != nil {
			return err
		}

		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) == ".json" || filepath.Ext(f.Name()) == ".jsonnet" {
				err = ReadJobDescriptions(filepath.Join(p, f.Name()), jobs)
				if err != nil {
					return err
				}
			}
		}
	} else {
		fd, err := os.Open(p)
		if err != nil {
			return err
		}
		defer fd.Close()

		err = ReadJobDescriptionsFile(p, fd, jobs)
		if err != nil {
			return err
		}
	}

	return nil
}

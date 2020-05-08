package ctnotify

import (
	ct "github.com/sloonz/cron-tools/pkg"

	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	flag "github.com/spf13/pflag"
)

func getDefaultTemplatesPath() string {
	usr, err := user.Current()
	if err == nil {
		usrPath := filepath.Join(usr.HomeDir, ".config", "cron-tools", "notify.d")
		_, err := os.Stat(usrPath)
		if err == nil {
			return usrPath
		}
	}

	sysPath := "/etc/cron-tools/notify.d"
	_, err = os.Stat(sysPath)
	if err == nil {
		return sysPath
	}

	return ""
}

func getDefaultTemplateFile() string {
	usr, err := user.Current()
	if err == nil {
		usrPath := filepath.Join(usr.HomeDir, ".config", "cron-tools", "notify.gotpl")
		_, err := os.Stat(usrPath)
		if err == nil {
			return usrPath
		}
	}

	sysPath := "/etc/cron-tools/notify.gotpl"
	_, err = os.Stat(sysPath)
	if err == nil {
		return sysPath
	}

	return ""
}

type LogLine struct {
	TimeOffset float32
	IsStderr   bool
	IsChunk    bool
	Data       string
}

type TemplateData struct {
	StartTime time.Time
	EndTime   time.Time
	Format    string // "LOG", "ERR", "INVALID"
	Headers   map[string]any
	Footers   map[string]any
	Lines     []LogLine
	RawData   string
}

func mustExecuteTemplate(tpl *template.Template, name string, data any) string {
	buf := bytes.NewBuffer(nil)
	err := tpl.ExecuteTemplate(buf, name, data)
	if err != nil {
		log.Fatal(err)
	}
	return buf.String()
}

func debugLogTemplate(tplID string, name string, value string) {
	if !ct.DebugEnabled() {
		return
	}

	log.Printf("%s: %s:\n%s", tplID, name, value)
}

func Main(args []string) {
	flags := flag.NewFlagSet("ct notify", flag.ExitOnError)
	configPathPtr := flags.StringP("templates-path", "p", getDefaultTemplatesPath(), "Path to templates directory")
	configFilePtr := flags.StringP("template-file", "f", getDefaultTemplateFile(), "Path to template file")
	verbosePtr := flags.BoolP("verbose", "V", false, "Verbose")
	_ = flags.Parse(args)

	templates := make(map[string]string)

	if *configPathPtr != "" {
		templateFiles, err := os.ReadDir(*configPathPtr)
		if err != nil {
			log.Fatal(err)
		}

		for i, fi := range templateFiles {
			if *verbosePtr {
				log.Printf("loading template _main%d: %s/%s", i, *configPathPtr, fi.Name())
			}
			if strings.HasSuffix(fi.Name(), ".gotpl") {
				templateString, err := os.ReadFile(filepath.Join(*configPathPtr, fi.Name()))
				if err != nil {
					log.Fatal(err)
				}

				templates[fmt.Sprintf("_main%d", i)] = string(templateString)
			}
		}
	}

	if *configFilePtr != "" {
		if *verbosePtr {
			log.Printf("loading template _main: %s", *configFilePtr)
		}
		templateString, err := os.ReadFile(*configFilePtr)
		if err != nil {
			log.Fatal(err)
		}

		templates["_main"] = string(templateString)
	}

	tplData := new(TemplateData)
	tplData.Format = "INVALID"
	tplData.Headers = make(map[string]any)
	tplData.Footers = make(map[string]any)
	tplData.Lines = make([]LogLine, 0, 1)

	rawBuf := bytes.NewBuffer(nil)
	ct.ParseLog(os.Stdin, ct.LogParser{
		OnFormat: func(timestamp time.Time, f string) {
			switch f {
			case ct.FormatLog:
				tplData.StartTime = timestamp
				tplData.Format = "LOG"
			case ct.FormatFailure:
				tplData.Format = "ERR"
			default:
				tplData.Format = "INVALID"
			}
		},
		OnMeta: func(timestamp time.Time, key, val string) {
			if key == "Exit" {
				tplData.EndTime = timestamp
				tplData.Footers[key] = val
			} else {
				tplData.Headers[key] = val
			}
		},
		OnOutput: func(timestamp time.Time, stream string, partial bool, data []byte) {
			var offset float32
			if !tplData.StartTime.IsZero() {
				offset = float32(timestamp.Sub(tplData.StartTime).Seconds())
			}
			tplData.Lines = append(tplData.Lines, LogLine{
				TimeOffset: offset,
				IsStderr:   stream == ct.CRIStreamStderr,
				IsChunk:    partial,
				Data:       string(data),
			})
		},
		OnRawOutput: func(data []byte) {
			if _, err := rawBuf.Write(data); err != nil {
				log.Fatal(err)
			}
		},
		OnError: func(err error) {
			_ = err
			tplData.Format = "INVALID"
		},
	})
	tplData.RawData = rawBuf.String()

	for tplID, tplString := range templates {
		var tpl *template.Template
		var err error

		tpl = template.New(tplID)
		tpl = template.Must(tpl.Funcs(ct.MakeTemplateFuncs(tpl)).Parse("{{define \"stdin\"}}{{end}}"))
		tpl, err = tpl.Parse(tplString)
		if err != nil {
			log.Fatal(err)
		}

		action := mustExecuteTemplate(tpl, "action", tplData)
		if *verbosePtr {
			log.Printf("%s: action: %s", tplID, action)
		}
		debugLogTemplate(tplID, "action", action)

		switch action {
		case "":
			// do nothing
		case "pipe":
			data := mustExecuteTemplate(tpl, "data", tplData)
			debugLogTemplate(tplID, "data", data)
			_, err = os.Stdout.WriteString(data)
			if err != nil {
				log.Fatal(err)
			}
		case "exec":
			command := mustExecuteTemplate(tpl, "command", tplData)
			stdin := mustExecuteTemplate(tpl, "stdin", tplData)
			if *verbosePtr {
				log.Printf("%s: command: %s", tplID, command)
			}
			debugLogTemplate(tplID, "command", command)
			debugLogTemplate(tplID, "stdin", stdin)

			cmd := exec.Command("/bin/sh", "-c", command)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = bytes.NewReader([]byte(stdin))

			err = cmd.Run()
			if err != nil {
				log.Fatalf("%s: %v", command, err)
			}
		default:
			log.Fatalf("unknown template action %s", action)
		}
	}
}

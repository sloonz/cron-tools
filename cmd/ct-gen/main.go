package ctgen

import (
	ct "github.com/sloonz/cron-tools/pkg"

	"bytes"
	"fmt"
	"log"
	"os"
	"text/template"

	flag "github.com/spf13/pflag"
)

func Main(args []string) {
	var signJob func(job ct.JobDescription) string

	flags := flag.NewFlagSet("ct gen", flag.ExitOnError)
	signingKeyPathPtr := flags.StringP("signing-key", "k", "", "Private signing key")
	templatePtr := flags.StringP("template", "t", "", "Template to use")
	userPtr := flags.StringP("user", "U", "", "If not empty, only process jobs of this user")
	hostPtr := flags.StringP("host", "H", "", "If not empty, only process jobs on this host")

	_ = flags.Parse(args)

	if *signingKeyPathPtr != "" {
		alg, err := ct.LoadJWTAlgorithmFromPrivateKeyFile(*signingKeyPathPtr)
		if err != nil {
			log.Fatal(err)
		}

		signJob = func(job ct.JobDescription) string {
			token, err := ct.SignJobToken(ct.Job{
				User:    job.User,
				Host:    job.Host,
				ID:      job.ID,
				Command: job.Command,
			}, alg)
			if err != nil {
				log.Fatal(err)
			}
			return string(token)
		}
	}

	if *templatePtr == "" {
		log.Fatal("no template provided")
	}

	var jobs []ct.JobDescription
	if len(flags.Args()) == 0 {
		err := ct.ReadJobDescriptionsFile("-", os.Stdin, &jobs)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		for _, p := range flags.Args() {
			err := ct.ReadJobDescriptions(p, &jobs)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	for _, job := range jobs {
		if (*userPtr != "" && *userPtr != job.User) || (*hostPtr != "" && *hostPtr != job.Host) {
			continue
		}

		buf := bytes.NewBuffer(nil)
		tpl := template.New("main")
		tpl = tpl.Funcs(ct.MakeTemplateFuncs(tpl)).Funcs(template.FuncMap{
			"sign": func(job ct.JobDescription) string {
				if signJob == nil {
					log.Fatal("no private key provided, cannot sign")
				}
				return signJob(job)
			},
		})
		tpl, err := tpl.Parse(*templatePtr)
		if err != nil {
			log.Fatal(err)
		}

		err = tpl.Execute(buf, job)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("# %s\n%s\n", job.ID, buf.String())
	}
}

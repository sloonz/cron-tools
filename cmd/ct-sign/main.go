package ctsign

import (
	ct "github.com/sloonz/cron-tools/pkg"

	flag "github.com/spf13/pflag"
	"log"
	"os"
)

func Main(args []string) {
	flags := flag.NewFlagSet("ct sign", flag.ExitOnError)
	userPtr := flags.StringP("user", "U", "", "User. If empty or omitted, the command can be run on all users.")
	hostPtr := flags.StringP("host", "H", "", "Host. If empty or omitted, the command can be run on all hosts.")
	idPtr := flags.StringP("id", "I", "", "Job ID")
	signingKeyPathPtr := flags.StringP("signing-key", "k", "", "Private signing key")

	_ = flags.Parse(args)

	if *idPtr == "" {
		log.Fatal("Job ID (--id/-I) is required")
	}

	if *signingKeyPathPtr == "" {
		log.Fatal("Private signing key (--signing-key/-k) required")
	}

	alg, err := ct.LoadJWTAlgorithmFromPrivateKeyFile(*signingKeyPathPtr)
	if err != nil {
		log.Fatal(err)
	}

	token, err := ct.SignJobToken(ct.Job{
		User:    *userPtr,
		Host:    *hostPtr,
		ID:      *idPtr,
		Command: flags.Args(),
	}, alg)

	if err != nil {
		log.Fatal(err)
	}

	if _, err := os.Stdout.Write(token); err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.WriteString("\n"); err != nil {
		log.Fatal(err)
	}
}

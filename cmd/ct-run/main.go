package ctrun

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/gbrlsnchs/jwt/v3"
	ct "github.com/sloonz/cron-tools/pkg"
	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"sync"
	"syscall"
	"time"
)

type lineBufferedWriter struct {
	stream string
	buf    []byte
	w      io.Writer

	// A buffered writer itself is not expected to be called from multiple goroutines
	// and don't need to be protected ; however, the underlying writer in general will
	// be shared between different goroutines and needs to be protected by a mutex to
	// avoid interleaving lines.
	mu *sync.Mutex
}

func (w *lineBufferedWriter) Write(data []byte) (int, error) {
	sz := len(data)
	data = append(w.buf, data...)
	lines := bytes.Split(data, []byte("\n"))
	if len(lines[len(lines)-1]) == 0 {
		w.buf = nil
	} else {
		w.buf = lines[len(lines)-1]
	}
	lines = lines[:len(lines)-1]
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, line := range lines {
		_, err := w.w.Write(formatBodyLine(w.stream, line, false))
		if err != nil {
			return 0, err
		}
	}
	return sz, nil
}

func (w *lineBufferedWriter) Close() error {
	if w.buf != nil {
		w.mu.Lock()
		defer w.mu.Unlock()
		_, err := w.w.Write(formatBodyLine(w.stream, w.buf, true))
		if err != nil {
			return err
		}
		w.buf = nil
	}
	return nil
}

func formatBodyLine(stream string, line []byte, isPartial bool) []byte {
	return ct.EncodeCRIOutputLine(time.Now().UTC(), stream, isPartial, line)
}

func writeMetaLine(timestamp time.Time, key, value string) {
	if _, err := os.Stdout.Write(ct.EncodeCRIMetaLine(timestamp, key, value)); err != nil {
		ct.Fatal("cannot write log event: %s", err.Error())
	}
}

func runJob(user string, host string, jobID string, command []string) {
	var stdoutMutex sync.Mutex

	cmdMarshaled, err := json.Marshal(command)
	if err != nil {
		ct.Fatal("%s: can't marshal command: %s", jobID, err.Error())
	}

	cmd := exec.Command(command[0], command[1:]...)
	stdout := &lineBufferedWriter{stream: ct.CRIStreamStdout, w: os.Stdout, mu: &stdoutMutex}
	stderr := &lineBufferedWriter{stream: ct.CRIStreamStderr, w: os.Stdout, mu: &stdoutMutex}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if _, err := os.Stdout.Write(ct.EncodeCRILine(time.Now().UTC(), ct.CRIStreamMeta, false, []byte(ct.FormatVersion))); err != nil {
		ct.Fatal("cannot write log marker: %s", err.Error())
	}
	writeMetaLine(time.Now().UTC(), "User", user)
	writeMetaLine(time.Now().UTC(), "Host", host)
	writeMetaLine(time.Now().UTC(), "Job-ID", jobID)
	writeMetaLine(time.Now().UTC(), "Command", string(cmdMarshaled))

	err = cmd.Run()
	endTime := time.Now().UTC()
	_ = stdout.Close()
	_ = stderr.Close()

	if err == nil {
		writeMetaLine(endTime, "Exit", "code 0")
	} else if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.Exited() {
			writeMetaLine(endTime, "Exit", fmt.Sprintf("code %d", exitError.ExitCode()))
		} else {
			status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
			if ok && status.Signaled() {
				writeMetaLine(endTime, "Exit", fmt.Sprintf("signal %s", status.Signal().String()))
			} else {
				writeMetaLine(endTime, "Exit", fmt.Sprintf("other %s", err.Error()))
			}
		}
	} else {
		writeMetaLine(endTime, "Exit", fmt.Sprintf("other %s", err.Error()))
	}
}

func Main(args []string) {
	host, err := os.Hostname()
	if err != nil {
		ct.Fatal("cannot find host: %s", err.Error())
	}

	usr, err := user.Current()
	if err != nil {
		ct.Fatal("%s: cannot find user: %s", host, err.Error())
	}

	flags := flag.NewFlagSet("ct run", flag.ExitOnError)
	jobIDPtr := flags.StringP("job-id", "I", "", "Job ID")
	signedJobPtr := flags.StringP("signed-job", "s", os.Getenv("SSH_ORIGINAL_COMMAND"), "Signed job")
	authorizedKeysPtr := flags.StringP("authorized-keys-file", "k", "", "Keys authorized to sign a job")
	_ = flags.Parse(args)

	if *signedJobPtr == "" {
		if *jobIDPtr == "" {
			ct.Fatal("job id is required")
		}

		if len(flags.Args()) == 0 {
			ct.Fatal("%s: command is required", *jobIDPtr)
		}

		runJob(usr.Username, host, *jobIDPtr, flags.Args())
	} else {
		var authorizedKeys []jwt.Algorithm

		for _, p := range []string{"/etc/cron-tools/authorized-keys", usr.HomeDir + "/.config/cron-tools/authorized-keys", *authorizedKeysPtr} {
			if _, err = os.Stat(p); err == nil {
				authorizedKeysData, err := os.ReadFile(p)
				if err != nil {
					ct.Fatal("%s\ncannot read authorized keys file %s", *signedJobPtr, p)
				}

				for {
					sshPubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysData)
					if err != nil {
						break
					}

					if sshCryptoPubKey, ok := sshPubKey.(ssh.CryptoPublicKey); ok {
						switch k := sshCryptoPubKey.CryptoPublicKey().(type) {
						case ed25519.PublicKey:
							authorizedKeys = append(authorizedKeys, jwt.NewEd25519(jwt.Ed25519PublicKey(k)))
						case *rsa.PublicKey:
							authorizedKeys = append(authorizedKeys, jwt.NewRS256(jwt.RSAPublicKey(k)))
						case *ecdsa.PublicKey:
							authorizedKeys = append(authorizedKeys, jwt.NewES256(jwt.ECDSAPublicKey(k)))
						default:
							ct.Fatal("%s\nunsupported public key type: %s", *signedJobPtr,
								reflect.TypeOf(sshCryptoPubKey).String())
						}
					} else {
						ct.Fatal("%s\nunspported public key", *signedJobPtr)
					}

					authorizedKeysData = rest
				}
			}
		}

		if authorizedKeys == nil {
			ct.Fatal("%s\nno key found", *signedJobPtr)
		}

		var job ct.Job
		validated := false
		for _, alg := range authorizedKeys {
			_, err = jwt.Verify([]byte(*signedJobPtr), alg, &job)
			if err == nil {
				validated = true
				break
			}
		}

		if !validated {
			ct.Fatal("%s\ncannot verify JWT token", *signedJobPtr)
		}

		if job.ID == "" {
			ct.Fatal("%s\nmissing job identifier", *signedJobPtr)
		}

		if job.User != "" && job.User != usr.Username {
			ct.Fatal("%s: invalid user", job.ID)
		}

		if job.Host != "" && job.Host != host {
			ct.Fatal("%s: invalid host", job.ID)
		}

		runJob(usr.Username, host, job.ID, job.Command)
	}
}

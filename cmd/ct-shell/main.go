package ctshell

import (
	"bytes"
	"fmt"
	ct "github.com/sloonz/cron-tools/pkg"
	"os"
	"os/exec"
	"sync"
)

type ctshellWriter struct {
	m            *sync.Mutex
	header       []byte
	headerDone   bool
	errorEmitted *bool
	isStdout     bool
}

func (w *ctshellWriter) emitError(err string) {
	// We want to emit the "CRONERR\n" magic header if:
	//  - it has not already been emitted AND
	//  - the first stdout line has not yet been emitted
	if !*w.errorEmitted && !w.headerDone {
		fmt.Printf("CRONERR\n")
	}

	fmt.Printf("ct shell: %s\n", err)
	*w.errorEmitted = true
}

func (w *ctshellWriter) Write(data []byte) (int, error) {
	w.m.Lock()
	defer w.m.Unlock()

	if w.isStdout {
		if !w.headerDone {
			w.header = append(w.header, data...)
			if newlinePos := bytes.IndexByte(w.header, '\n'); newlinePos >= 0 {
				firstLine := string(w.header[:newlinePos+1])
				if firstLine != ct.FormatFailure {
					timestamp, stream, partial, message, err := ct.ParseCRILine(firstLine)
					_ = timestamp
					if err != nil || stream != ct.CRIStreamMeta || partial || message != ct.FormatVersion {
						if !*w.errorEmitted {
							*w.errorEmitted = true
							fmt.Printf("CRONERR\n")
						}
						fmt.Printf("ct shell: invalid header\n")
					}
				}
				if _, err := os.Stdout.Write(w.header); err != nil {
					return len(data), err
				}
				w.header = nil
				w.headerDone = true
			}
			return len(data), nil
		}
	} else {
		w.emitError("non-empty stderr")
	}

	_, err := os.Stdout.Write(data)

	return len(data), err
}

func Main(args []string) {
	var command []string

	switch {
	case len(args) == 0:
		ct.Fatal("invalid arguments")
	case args[0] == "-c":
		if len(args) != 2 {
			ct.Fatal("invalid arguments")
		}
		command = append(command, "/bin/sh", "-c", args[1])
	default:
		command = args
	}

	var mutex sync.Mutex
	errorEmitted := false

	stdout := &ctshellWriter{m: &mutex, errorEmitted: &errorEmitted, isStdout: true, header: nil}
	stderr := &ctshellWriter{m: &mutex, errorEmitted: &errorEmitted, isStdout: false, header: nil}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()

	if !stdout.headerDone {
		stdout.emitError(fmt.Sprintf("stdout: unexpected end of file (output was: %s)", string(stdout.header)))
	}

	if err != nil {
		stdout.emitError(fmt.Sprintf("%v: %s", command, err.Error()))
	}

	if errorEmitted {
		os.Exit(1)
	}
}

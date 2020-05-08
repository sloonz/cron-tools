package crontools

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"time"
)

const FormatInvalid = ""
const FormatFailure = "CRONERR\n"
const FormatLog = "CRI\n"
const FormatVersion = "cronlog-v0"

const (
	CRIStreamStdout = "stdout"
	CRIStreamStderr = "stderr"
	CRIStreamMeta   = "meta"
)

type LogParser struct {
	OnFormat    func(timestamp time.Time, format string)
	OnMeta      func(timestamp time.Time, key, val string)
	OnOutput    func(timestamp time.Time, stream string, partial bool, data []byte)
	OnRawOutput func(data []byte)
	OnClose     func()
	OnError     func(err error)
}

func ParseCRILine(line string) (time.Time, string, bool, string, error) {
	line = strings.TrimSuffix(line, "\n")
	parts := strings.SplitN(line, " ", 4)
	if len(parts) != 4 {
		return time.Time{}, "", false, "", errors.New("parse error: cannot split CRI line")
	}

	timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false, "", errors.New("parse error: invalid timestamp")
	}

	stream := parts[1]
	if stream != CRIStreamStdout && stream != CRIStreamStderr && stream != CRIStreamMeta {
		return time.Time{}, "", false, "", errors.New("parse error: invalid stream")
	}

	var partial bool
	switch parts[2] {
	case "F":
		partial = false
	case "P":
		partial = true
	default:
		return time.Time{}, "", false, "", errors.New("parse error: invalid partial flag")
	}

	return timestamp.UTC(), stream, partial, parts[3], nil
}

func EncodeCRILine(timestamp time.Time, stream string, partial bool, message []byte) []byte {
	flag := "F"
	if partial {
		flag = "P"
	}

	return []byte(timestamp.UTC().Format(time.RFC3339Nano) + " " + stream + " " + flag + " " + string(message) + "\n")
}

func EncodeCRIMetaLine(timestamp time.Time, key, value string) []byte {
	return EncodeCRILine(timestamp, CRIStreamMeta, false, []byte(key+": "+value))
}

func EncodeCRIOutputLine(timestamp time.Time, stream string, partial bool, data []byte) []byte {
	return EncodeCRILine(timestamp, stream, partial, data)
}

func ParseLog(log io.Reader, parser LogParser) {
	r := bufio.NewReader(log)

	onError := func(err error) {
		if parser.OnError != nil {
			parser.OnError(err)
		}
	}
	onRawOutput := func(data []byte) {
		if parser.OnRawOutput != nil {
			parser.OnRawOutput(data)
		}
	}
	onClose := func() {
		if parser.OnClose != nil {
			parser.OnClose()
		}
	}
	onFormat := func(timestamp time.Time, format string) {
		if parser.OnFormat != nil {
			parser.OnFormat(timestamp, format)
		}
	}
	readRest := func() {
		buf := make([]byte, 1024)
		for {
			sz, err := r.Read(buf)
			if err != nil {
				if err != io.EOF {
					onError(err)
				}
				break
			}
			onRawOutput(buf[:sz])
		}
		onClose()
	}

	firstLine, err := r.ReadString('\n')
	if err == io.EOF && firstLine != "" {
		onRawOutput([]byte(firstLine))
		onFormat(time.Time{}, FormatInvalid)
		onError(io.EOF)
		onClose()
		return
	}
	if err != nil {
		onError(err)
		onClose()
		return
	}

	onRawOutput([]byte(firstLine))
	if firstLine == FormatFailure {
		onFormat(time.Time{}, FormatFailure)
		readRest()
		return
	}
	firstRecord := true

	for {
		timestamp, stream, partial, message, err := ParseCRILine(firstLine)
		if err != nil {
			onFormat(time.Time{}, FormatInvalid)
			onError(err)
			readRest()
			return
		}

		if firstRecord {
			firstRecord = false
			if stream != CRIStreamMeta || partial || message != FormatVersion {
				onFormat(timestamp, FormatInvalid)
				onError(errors.New("parse error: missing cronlog marker"))
				readRest()
				return
			}
			onFormat(timestamp, FormatLog)
		} else {
			switch stream {
			case CRIStreamMeta:
				parts := strings.SplitN(message, ":", 2)
				if len(parts) != 2 {
					onFormat(timestamp, FormatInvalid)
					onError(errors.New("parse error: no metadata key/value separator found"))
					readRest()
					return
				}
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])

				if parser.OnMeta != nil {
					parser.OnMeta(timestamp, key, val)
				}

			case CRIStreamStdout, CRIStreamStderr:

				if parser.OnOutput != nil {
					parser.OnOutput(timestamp, stream, partial, []byte(message))
				}
			}

		}

		firstLine, err = r.ReadString('\n')
		if err == io.EOF {
			onClose()
			return
		}
		if err != nil {
			onError(err)
			onClose()
			return
		}
		onRawOutput([]byte(firstLine))
	}
}

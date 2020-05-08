package crontools

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
	"time"

	"al.essio.dev/pkg/shellescape"
	"github.com/Masterminds/sprig/v3"
)

func MakeTemplateFuncs(tpl *template.Template) template.FuncMap {
	funcMap := map[string]any{
		"timeFormats": func() map[string]string {
			return map[string]string{
				"Layout":      time.Layout,
				"ANSIC":       time.ANSIC,
				"UnixDate":    time.UnixDate,
				"RubyDate":    time.RubyDate,
				"RFC822":      time.RFC822,
				"RFC822Z":     time.RFC822Z,
				"RFC850":      time.RFC850,
				"RFC1123":     time.RFC1123,
				"RFC1123Z":    time.RFC1123Z,
				"RFC3339":     time.RFC3339,
				"RFC3339Nano": time.RFC3339Nano,
				"Kitchen":     time.Kitchen,
				"Stamp":       time.Stamp,
				"StampMilli":  time.StampMilli,
				"StampMicro":  time.StampMicro,
				"StampNano":   time.StampNano,
				"DateTime":    time.DateTime,
				"DateOnly":    time.DateOnly,
				"TimeOnly":    time.TimeOnly,
			}
		},
		"eval": func(name string, data any) (string, error) {
			buf := bytes.NewBuffer(nil)
			err := tpl.ExecuteTemplate(buf, name, data)
			return buf.String(), err
		},
		"hostname": func() (string, error) {
			return os.Hostname()
		},
		"readfile": func(path string) (string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
		"shescape": shellescape.Quote,
		"shjoin": func(argsIface any) (string, error) {
			if args, ok := argsIface.([]string); ok {
				return shellescape.QuoteCommand(args), nil
			} else if args, ok := argsIface.([]any); ok {
				strArgs := make([]string, 0, len(args))
				for _, arg := range args {
					strArg, ok := arg.(string)
					if ok {
						strArgs = append(strArgs, strArg)
					} else {
						return "", fmt.Errorf("invalid argument to shjoin: %v", arg)
					}
				}
				return shellescape.QuoteCommand(strArgs), nil
			}

			return "", fmt.Errorf("invalid argument to shjoin: %v", argsIface)
		},
	}
	for k, v := range sprig.GenericFuncMap() {
		funcMap[k] = v
	}
	for k, v := range sprig.HermeticTxtFuncMap() {
		funcMap[k] = v
	}
	return funcMap
}

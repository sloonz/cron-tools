package crontools

import (
	"fmt"
	"os"
	"time"
)

func Fatal(format string, args ...any) {
	if _, err := os.Stdout.WriteString("CRONERR\n"); err != nil {
		fmt.Fprintf(os.Stderr, "cannot write error header: %v\n", err)
		os.Exit(1)
	}
	args = append([]any{time.Now().UTC().Format(time.RFC3339)}, args...)
	fmt.Printf("%s "+format+"\n", args...)
	os.Exit(1)
}

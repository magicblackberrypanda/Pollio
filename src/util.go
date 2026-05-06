package main

import (
	"fmt"
	"os"
	"time"
)

func timestamp() string {
	return time.Now().Format(time.RFC3339)
}

func debugf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "%s [DEBUG] %s\n", timestamp(), fmt.Sprintf(format, v...))
}

func infof(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "%s [INFO] %s\n", timestamp(), fmt.Sprintf(format, v...))
}

func errorf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "%s [ERROR] %s\n", timestamp(), fmt.Sprintf(format, v...))
	os.Exit(1)
}

func warningf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "%s [WARNING] %s\n", timestamp(), fmt.Sprintf(format, v...))
}

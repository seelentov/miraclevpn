// Package utils provides utility functions
package utils

import (
	"fmt"
	"runtime"
)

func GetStackTrace(err error) string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	stack := string(buf[:n])

	return fmt.Sprintf("Error: %v\nStack Trace:\n%s", err, stack)
}

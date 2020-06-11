package gotemplate

import (
	"fmt"
	"io"
)

// Print takes a message and prints it to the specified io.Writer.
func Print(w io.Writer, message string) {
	fmt.Fprint(w, message, "\n")
}

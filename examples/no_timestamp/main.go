package main

import (
	"bytes"
	"strings"

	"github.com/alexjoedt/log"
)

func main() {
	var buf bytes.Buffer

	// Create logger without timestamp
	logger := log.New(
		log.WithWriter(&buf),
		log.WithFormat(log.FormatJSON),
		log.WithoutTimestamp(),
	)

	logger.Info("This message should not have a timestamp")

	output := buf.String()
	println("Output:", output)

	// Verify that the output doesn't start with a timestamp-like pattern (YYYY-MM-DD)
	if strings.Contains(output, "2") && strings.Contains(output, ":") {
		println("WARNING: Output might still contain timestamp!")
	} else {
		println("✓ Success: No timestamp in output!")
	}
}

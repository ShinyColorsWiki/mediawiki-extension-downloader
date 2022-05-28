package main

import (
	"os"
	"testing"
)

func TestMainProgram(t *testing.T) {
	os.Args = []string{"program",
		"--config", "./config.json.example",
		"--target", "/tmp/extension-download"}
	main()

	if hasError {
		log.Child("TEST").Fatal("Error during test")
	}
}

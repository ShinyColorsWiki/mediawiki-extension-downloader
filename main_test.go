package main

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func TestMainProgram(t *testing.T) {
	tempdir := fmt.Sprintf("/tmp/mediawiki-extension-downloader-test-%d", rand.Int())

	os.Args = []string{"program",
		"--config", "./config.example.json",
		"--target", tempdir}
	main()

	// cleanup tmp directory
	err := os.RemoveAll(tempdir)
	if err != nil {
		log.Child("TEST").Error("Unknown error occured during cleanup. This doesn't affect the test. ", err)
	}

	if hasError {
		log.Child("TEST").Fatal("Error during test")
	}
}

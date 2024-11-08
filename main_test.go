package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMainProgram(t *testing.T) {
	// Create a temporary directory for the test using t.TempDir()
	tempDir := t.TempDir()

	// Set up command-line arguments for the main program.
	os.Args = []string{
		"program",
		"--config", "./config.example.json",
		"--target", tempDir,
	}

	// Run the main program.
	main()

	// Read and validate the configuration file.
	config, err := readConfig("./config.example.json")
	if err != nil {
		t.Fatalf("Error reading config: %v", err)
	}

	// Calculate the total number of extensions and skins.
	extensionsLen := len(config.Extensions.WMF) + len(config.Extensions.Git) + len(config.Extensions.Http)
	if extensionsLen == 0 {
		t.Fatal("No extensions found in config file.")
	}

	skinsLen := len(config.Skins.WMF) + len(config.Skins.Git) + len(config.Skins.Http)
	if skinsLen == 0 {
		t.Fatal("No skins found in config file.")
	}

	t.Log("Config file read successfully.")

	// Verify the extensions directory.
	extensionsDir := filepath.Join(tempDir, "extensions")
	extEntries, err := os.ReadDir(extensionsDir)
	if err != nil {
		t.Fatalf("Error reading extensions directory: %v", err)
	}
	t.Logf("Extensions found: %v", extEntries)
	t.Logf("Expected number of extensions: %d", extensionsLen)

	if len(extEntries) == 0 || len(extEntries) != extensionsLen {
		t.Fatalf("Extensions directory is empty or does not match config. Expected: %d, Found: %d", extensionsLen, len(extEntries))
	}

	// Ensure each extension directory contains at least two files.
	for _, entry := range extEntries {
		extPath := filepath.Join(extensionsDir, entry.Name())
		files, err := os.ReadDir(extPath)
		if err != nil {
			t.Fatalf("Error reading extension directory '%s': %v", extPath, err)
		}
		if len(files) < 2 {
			t.Fatalf("Extension directory '%s' should have at least 2 files. Found: %d", extPath, len(files))
		}
	}

	// Verify the skins directory.
	skinsDir := filepath.Join(tempDir, "skins")
	skinEntries, err := os.ReadDir(skinsDir)
	if err != nil {
		t.Fatalf("Error reading skins directory: %v", err)
	}
	t.Logf("Skins found: %v", skinEntries)
	t.Logf("Expected number of skins: %d", skinsLen)

	if len(skinEntries) == 0 || len(skinEntries) != skinsLen {
		t.Fatalf("Skins directory is empty or does not match config. Expected: %d, Found: %d", skinsLen, len(skinEntries))
	}

	// Ensure each skin directory contains at least two files.
	for _, entry := range skinEntries {
		skinPath := filepath.Join(skinsDir, entry.Name())
		files, err := os.ReadDir(skinPath)
		if err != nil {
			t.Fatalf("Error reading skin directory '%s': %v", skinPath, err)
		}
		if len(files) < 2 {
			t.Fatalf("Skin directory '%s' should have at least 2 files. Found: %d", skinPath, len(files))
		}
	}

	// Final check to ensure the temporary directory exists.
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Fatalf("Temporary directory not found: %s", tempDir)
	}
}

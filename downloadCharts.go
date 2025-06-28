package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Chart struct {
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
	Chart   string `yaml:"chart"`
	Path    string `yaml:"path"`
	Version string `yaml:"version"`
}

type ChartFile struct {
	Charts []Chart `yaml:"charts"`
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func folderExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func copyDir(src string, dst string) error {
	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			in, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.Create(dstPath)
			if err != nil {
				return err
			}
			defer out.Close()
			if _, err := io.Copy(out, in); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	data, err := ioutil.ReadFile("charts.yaml")
	if err != nil {
		log.Fatalf("Failed to read charts.yaml: %v", err)
	}

	var chartFile ChartFile
	if err := yaml.Unmarshal(data, &chartFile); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	addedRepos := make(map[string]bool)

	for _, chart := range chartFile.Charts {
		// Determine folder name: path if given, else chart
		folderName := chart.Path
		if folderName == "" {
			folderName = chart.Chart
		}

		// Remove folder if exists
		if folderExists(folderName) {
			fmt.Printf("Removing existing folder: %s\n", folderName)
			if err := os.RemoveAll(folderName); err != nil {
				log.Fatalf("Failed to remove folder: %v", err)
			}
		}

		// Add repo if not already added
		if !addedRepos[chart.Name] {
			fmt.Printf("Adding repo: %s -> %s\n", chart.Name, chart.URL)
			if err := runCmd("helm", "repo", "add", chart.Name, chart.URL); err != nil {
				log.Fatalf("Failed to add repo %s: %v", chart.Name, err)
			}
			addedRepos[chart.Name] = true
		}

		// Update repos
		fmt.Println("Updating helm repos...")
		if err := runCmd("helm", "repo", "update"); err != nil {
			log.Fatalf("Failed to update repos: %v", err)
		}

		// Use a temp directory for untar
		tmpDir, err := ioutil.TempDir("", "helm-pull")
		if err != nil {
			log.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		fmt.Printf("Pulling chart: %s/%s into temp dir\n", chart.Name, chart.Chart)
		if err := runCmd("helm", "pull",
			fmt.Sprintf("%s/%s", chart.Name, chart.Chart),
			"--version", chart.Version,
			"--untar", "--untardir", tmpDir); err != nil {
			log.Fatalf("Failed to pull chart: %v", err)
		}

		// Move contents from tmpDir/chart.Chart to folderName
		srcDir := filepath.Join(tmpDir, chart.Chart)
		if err := copyDir(srcDir, folderName); err != nil {
			log.Fatalf("Failed to copy chart to destination: %v", err)
		}
	}
}

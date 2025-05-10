package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"gopkg.in/yaml.v2"
)

type Chart struct {
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
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
		// Remove folder if exists
		if folderExists(chart.Path) {
			fmt.Printf("Removing existing folder: %s\n", chart.Path)
			if err := os.RemoveAll(chart.Path); err != nil {
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

		// Update repos (optional, but good to keep charts fresh)
		fmt.Println("Updating helm repos...")
		if err := runCmd("helm", "repo", "update"); err != nil {
			log.Fatalf("Failed to update repos: %v", err)
		}

		// Pull the chart
		fmt.Printf("Pulling chart: %s/%s\n", chart.Name, chart.Path)
		if err := runCmd("helm", "pull",
			fmt.Sprintf("%s/%s", chart.Name, chart.Path),
			"--version", chart.Version,
			"--untar", "--untardir", "."); err != nil {
			log.Fatalf("Failed to pull chart: %v", err)
		}
	}
}

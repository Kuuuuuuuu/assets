package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// yay, finally I know how to use constants and variables in go LOL
const (
	dataFilePath     = "data.json"
	imagesFolderPath = "images"
	githubAPIURL     = "https://api.github.com/repos/%s/%s/languages"
	githubImageURL   = "https://opengraph.githubassets.com/main/%s/%s"
)

var (
	githubRegex = regexp.MustCompile(`https://github.com/([^/]+)/([^/]+)`)
)

type Value struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Image       string   `json:"image"`
	Link        string   `json:"link"`
	Status      string   `json:"status,omitempty"`
	Languages   []string `json:"languages,omitempty"`
}

type Data map[string]Value

func main() {
	data, err := readDataFromFile(dataFilePath)
	if err != nil {
		log.Fatalf("Error reading data file: %v", err)
	}

	updateData(data)

	if err := dataToFile(data, dataFilePath); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	updateReadme()
}

func readDataFromFile(filePath string) (Data, error) {
	dataFile, err := os.ReadFile(filePath)
	if err != nil {
		return Data{}, fmt.Errorf("error reading data file: %w", err)
	}

	var data Data
	if err := json.Unmarshal(dataFile, &data); err != nil {
		return Data{}, fmt.Errorf("error parsing data: %w", err)
	}

	return data, nil
}

func updateData(data Data) {
	for key, value := range data {
		if githubRegex.MatchString(value.Link) {
			matches := githubRegex.FindStringSubmatch(value.Link)
			if len(matches) == 3 {
				owner, repo := matches[1], matches[2]
				// not sure is this good but It's working so I'll keep it lol
				imagePath := filepath.Join(imagesFolderPath, fmt.Sprintf("%s.png", repo))
				if err := downloadImage(owner, repo, imagePath); err != nil {
					log.Printf("Error downloading image for %s: %v\n", value.Name, err)
					continue
				}

				// get the updated value
				updatedValue := getDataFromRepo(value, owner, repo)
				data[key] = updatedValue
			}
		}
	}
}

func downloadImage(owner, repo, filePath string) error {
	// Check if the image already exists. if it does, delete it
	// because we want image always up to date
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf(githubImageURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error downloading image: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image. Status: %d", response.StatusCode)
	}

	// temp file in case something error
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), "temp-image-*")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	if _, err = io.Copy(tempFile, response.Body); err != nil {
		tempFile.Close()
		return fmt.Errorf("error saving image file: %w", err)
	}

	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("error closing temporary file: %w", err)
	}

	if err = os.Rename(tempFilePath, filePath); err != nil {
		return fmt.Errorf("error replacing old image with new one: %w", err)
	}

	return nil
}

func getDataFromRepo(value Value, owner string, repo string) Value {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf(githubAPIURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return value
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error fetching data from %s: %v\n", url, err)
		return value
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Printf("Failed to fetch data from %s. Status: %d", url, response.StatusCode)
		return value
	}

	var jsonMap map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&jsonMap); err != nil {
		log.Printf("Error parsing JSON data: %v", err)
		return value
	}

	value.Languages = make([]string, 0, len(jsonMap))
	for key := range jsonMap {
		value.Languages = append(value.Languages, key)
	}

	return value
}

func dataToFile(data Data, filePath string) error {
	buffer := &bytes.Buffer{}

	encoder := json.NewEncoder(buffer)
	encoder.SetIndent("", "  ") // 2 spaces :D
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("error while encoding data: %w", err)
	}

	if err := os.WriteFile(filePath, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

func updateReadme() {
	location, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		log.Fatalf("Error loading location: %v", err)
	}

	currentDate := time.Now().In(location).Format("2006-01-02 15:04:05")

	readme, err := os.ReadFile("README.md")
	if err != nil {
		log.Fatalf("Error reading README.md: %v", err)
	}

	re := regexp.MustCompile(`Last Updated: \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	updatedReadme := re.ReplaceAllString(string(readme), "Last Updated: "+currentDate)

	if err := os.WriteFile("README.md", []byte(updatedReadme), 0644); err != nil {
		log.Fatalf("Error writing to README.md: %v", err)
	}

	log.Println("Updated README.md")
}

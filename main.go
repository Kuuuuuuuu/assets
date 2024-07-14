package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
		return
	}

	updateData(data)

	if err := dataToFile(data, dataFilePath); err != nil {
		log.Fatalf("Error writing to file: %v", err)
		return
	}

	updateReadme()
}

func readDataFromFile(filePath string) (Data, error) {
	dataFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading data file: %v", err)
	}

	var data Data
	if err := json.Unmarshal(dataFile, &data); err != nil {
		return nil, fmt.Errorf("error parsing data: %v", err)
	}

	return data, nil
}

func updateData(data Data) {
	for key, value := range data {
		if githubRegex.MatchString(value.Link) {
			matches := githubRegex.FindStringSubmatch(value.Link)
			if len(matches) == 3 {
				// not sure is this good but It's working so I'll keep it lol
				imagePath := filepath.Join(imagesFolderPath, fmt.Sprintf("%s.png", matches[2]))
				if err := downloadImage(matches[1], matches[2], imagePath); err != nil {
					log.Printf("Error downloading image for %s: %v\n", value.Name, err)
					continue
				}

				// get the updated value
				updatedValue := getDataFromRepo(value, matches[1], matches[2])
				data[key] = updatedValue
			}
		}
	}
}

func downloadImage(owner, repo, filePath string) error {
	// Check if the image already exists. if it does, delete it
	// because we want image always up to date
	client := &http.Client{
		Timeout: 10 * time.Second, // 10 s timeout for idk aaaaaa
	}

	url := fmt.Sprintf(githubImageURL, owner, repo)
	response, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading image: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image. Status: %d", response.StatusCode)
	}

	if _, err := os.Stat(filePath); err == nil {
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("error deleting existing image: %v", err)
		}
	}

	imageFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating image file: %v", err)
	}
	defer func() { // handle the error if the file is not closed properly (I think)
		if closeErr := imageFile.Close(); closeErr != nil {
			log.Printf("Error closing image file: %v", closeErr)
		}
	}()

	if _, err = io.Copy(imageFile, response.Body); err != nil {
		return fmt.Errorf("error saving image file: %v", err)
	}

	return nil
}

func getDataFromRepo(value Value, owner string, repo string) Value {
	client := &http.Client{
		Timeout: 10 * time.Second, // 10 s timeout for idk aaaaaa
	}

	url := fmt.Sprintf(githubAPIURL, owner, repo)
	response, err := client.Get(url)
	if err != nil {
		log.Printf("Error fetching data from %s: %v", url, err)
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

	languages := make([]string, 0, len(jsonMap))
	for key := range jsonMap {
		languages = append(languages, key)
	}

	value.Languages = languages
	return value
}

func dataToFile(data Data, filePath string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error while marshalling data: %v", err)
	}

	// replace \u0026 with & in the JSON data still don't know to properly handle this LOL
	replacer := strings.NewReplacer(`\u0026`, `&`)
	jsonData = []byte(replacer.Replace(string(jsonData)))

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	return nil
}

func updateReadme() {
	// it not my fault that idk about LoadLocation function LOL
	location, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		log.Fatalf("Error loading location: %v", err)
		return
	}

	currentDate := time.Now().In(location).Format("2006-01-02 15:04:05")

	readme, err := os.ReadFile("README.md")
	if err != nil {
		log.Fatalf("Error reading README.md: %v", err)
		return
	}

	// don't mind my code lol trying to learn go
	re := regexp.MustCompile(`Last Updated: \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	updatedReadme := re.ReplaceAllString(string(readme), "Last Updated: "+currentDate)

	err = os.WriteFile("README.md", []byte(updatedReadme), 0644)
	if err != nil {
		log.Fatalf("Error writing to README.md: %v", err)
		return
	}

	log.Println("Updated README.md")
}

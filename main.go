package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
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

const dataFilePath = "data.json"

func main() {
	data, err := readDataFromFile(dataFilePath)
	if err != nil {
		log.Fatalf("Error: %v", err)
		return
	}

	updateLanguages(data)

	if err := dataToFile(data, dataFilePath); err != nil {
		log.Fatalf("Error: %v", err)
		return
	}

	updateReadme()
}

func readDataFromFile(filePath string) (Data, error) {
	dataFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Error reading data file: %v", err)
	}

	var data Data
	if err := json.Unmarshal(dataFile, &data); err != nil {
		return nil, fmt.Errorf("Error parsing data: %v", err)
	}

	return data, nil
}

func updateLanguages(data Data) {
	for key, value := range data {
		// this might better for github username and repo name
		githubRegex := regexp.MustCompile(`https://github.com/([^/]+)/([^/]+)`)
		if githubRegex.MatchString(value.Link) {
			matches := githubRegex.FindStringSubmatch(value.Link)
			if len(matches) == 3 {
				// get the updated value
				updatedValue := getDataFromRepo(value, matches[1], matches[2])
				data[key] = updatedValue
			}
		}
	}
}

func getDataFromRepo(value Value, owner, repo string) Value {
	client := &http.Client{} // wtf I never knew this is a thing

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/languages", owner, repo)
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

	var languages []string
	for key := range jsonMap {
		languages = append(languages, key)
	}

	value.Languages = languages
	return value
}

func dataToFile(data Data, filePath string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("Error while marshalling data: %v", err)
	}

	// replace \u0026 with & in the JSON data still don't know to properly handle this LOL
	jsonData = bytes.ReplaceAll(jsonData, []byte(`\u0026`), []byte(`&`))

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("Error writing to file: %v", err)
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

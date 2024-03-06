package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
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
		fmt.Printf("Error: %v\n", err)
		return
	}

	updateLanguages(data)

	if err := dataToFile(data, dataFilePath); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
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
		githubRegex := regexp.MustCompile(`https:\/\/github.com\/([a-zA-Z0-9-]+)\/([a-zA-Z0-9-]+)`)
		if githubRegex.MatchString(value.Link) {
			matches := githubRegex.FindStringSubmatch(value.Link)
			if len(matches) == 3 {
				getDataFromRepo(&value, matches[1], matches[2])
				data[key] = value
			}
		}
	}
}

func getDataFromRepo(value *Value, owner, repo string) {
	response, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/languages", owner, repo))
	if err != nil {
		fmt.Printf("Error fetching data: %v\n", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Printf("Failed to fetch data. Status: %d\n", response.StatusCode)
		return
	}

	var jsonMap map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&jsonMap); err != nil {
		fmt.Printf("Error parsing JSON data: %v\n", err)
		return
	}

	var languages []string
	for key := range jsonMap {
		languages = append(languages, key)
	}

	value.Languages = languages
}

func dataToFile(data Data, filePath string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("Error while marshalling data: %v", err)
	}

	// replace \u0026 with & in the JSON data idk how to properly handle this lol
	jsonData = []byte(strings.ReplaceAll(string(jsonData), `\u0026`, `&`))

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("Error writing to file: %v", err)
	}

	return nil
}

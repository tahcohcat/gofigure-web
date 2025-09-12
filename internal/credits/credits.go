package credits

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Credit struct {
	ImagePath  string        `json:"image_path"`
	CreditHTML template.HTML `json:"credit_html"`
}

type MysteryData struct {
	Credits []Credit `json:"credits"`
}

func Handler(w http.ResponseWriter, _ *http.Request) {
	// Path to the mysteries directory
	dirPath := "./data/mysteries"

	// Read all files from the directory
	files, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, "Failed to read mysteries directory", http.StatusInternalServerError)
		log.Printf("Error reading directory %s: %v", dirPath, err)
		return
	}

	// Use a map to collect unique credits, keyed by the image path
	allCredits := make(map[string]Credit)

	for _, file := range files {
		// Process only JSON files
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(dirPath, file.Name())

			// Read file content
			data, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("Warning: could not read mystery file %s: %v", filePath, err)
				continue // Skip this file
			}

			// Unmarshal into our struct
			var mystery MysteryData
			if err := json.Unmarshal(data, &mystery); err != nil {
				log.Printf("Warning: could not unmarshal mystery file %s: %v", filePath, err)
				continue // Skip this file
			}

			// Handler credits to our map to ensure uniqueness
			for _, credit := range mystery.Credits {
				allCredits[credit.ImagePath] = credit
			}
		}
	}

	// Convert map to a slice for templating
	uniqueCredits := make([]Credit, 0, len(allCredits))
	for _, credit := range allCredits {
		uniqueCredits = append(uniqueCredits, credit)
	}

	// Sort the credits alphabetically by image path for consistent ordering
	sort.Slice(uniqueCredits, func(i, j int) bool {
		return uniqueCredits[i].ImagePath < uniqueCredits[j].ImagePath
	})

	// Parse and execute the template
	tmpl, err := template.ParseFiles("./web/templates/credits.gohtml")
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		log.Printf("Error parsing template: %v", err)
		return
	}

	if err := tmpl.Execute(w, uniqueCredits); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
	}
}

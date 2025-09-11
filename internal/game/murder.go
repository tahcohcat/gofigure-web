package game

import (
	"github.com/schollz/closestmatch"
)

// Murder scenario loaded from JSON
type Murder struct {
	Title       string      `json:"title"`
	Killer      string      `json:"killer"`
	Weapon      string      `json:"weapon"`
	Location    string      `json:"location"`
	Intro       string      `json:"introduction"`
	NarratorTTS []TTS       `json:"narrator_tts,omitempty"`
	Characters  []Character `json:"characters"`
}

func (m *Murder) closesCharacterMatches() *closestmatch.ClosestMatch {
	names := []string{}
	for _, char := range m.Characters {
		names = append(names, char.Name)
	}

	return closestmatch.New(names, []int{2})
}

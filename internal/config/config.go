package config

import (
	"encoding/json"
	"os"
)

type EventColor int

const (
	Lavender = iota + 1
	Sage
	Grape
	Flamingo
	Banana
	Tangerine
	Peacock
	Graphite
	Blueberry
	Basil
	Tomato
)

type CalendarServiceOptions struct {
	TwoWaySync          bool   `json:"twoWaySync"`
	RedactedSummary     bool   `json:"redactedSummary"`
	RedactedDescription bool   `json:"redactedDescription"`
	RedactedLocation    bool   `json:"redactedLocation"`
	RedactedAttendees   bool   `json:"redactedAttendees"`
	RedactedOrganizer   bool   `json:"redactedOrganizer"`
	RedactedAtachments  bool   `json:"redactedAtachments"`
	DisableReminders    bool   `json:"disableReminders"`
	EventColor          string `json:"eventColor"`
}

type Config struct {
	SyncOptions CalendarServiceOptions `json:"syncOptions"`

	Source                string `json:"source"`
	SourceCredentialsFile string `json:"sourceCredentialsFile"`

	Destination                string `json:"destination"`
	DestinationCredentialsFile string `json:"destinationCredentialsFile"`

	CallbackServerPort int `json:"callbackServerPort"`
}

func LoadConfiguration(file string) (Config, error) {
	var config Config
	configFile, err := os.ReadFile(file)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(configFile, &config)
	return config, err
}

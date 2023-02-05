package internal

import (
	"log"

	"github.com/marianozunino/go-sync-cal/internal/config"
	"github.com/marianozunino/go-sync-cal/internal/service"
)

func Run() {
	config, err := config.LoadConfiguration("config.json")
	if err != nil {
		log.Fatalf("Unable to load configuration: %v", err)
	}

	calendarService := service.CalendarService{}
	calendarService.Run(config)
}

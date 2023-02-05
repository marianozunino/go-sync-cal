package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/marianozunino/go-sync-cal/internal/config"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type CalendarService struct {
	options          config.Config
	vendorClientMap  map[string]*http.Client
	vendorServiceMap map[string]*calendar.Service
	vendorEvents     map[string][]*calendar.Event
}

func (c *CalendarService) Run(options config.Config) {
	c.options = options

	c.vendorClientMap = make(map[string]*http.Client)
	c.vendorServiceMap = make(map[string]*calendar.Service)
	c.vendorEvents = make(map[string][]*calendar.Event)

	c.setupCalendarClients(c.options.Source, c.options.SourceCredentialsFile)
	c.setupCalendarClients(c.options.Destination, c.options.DestinationCredentialsFile)

	if c.options.SyncOptions.TwoWaySync {
		c.loadVendorEvents(c.options.Source)
		c.loadVendorEvents(c.options.Destination)
	} else {
		c.loadVendorEvents(c.options.Source)
	}

	c.importEvents(c.options.Source, c.options.Destination)
	if c.options.SyncOptions.TwoWaySync {
		c.importEvents(c.options.Destination, c.options.Source)
	}
}

func (c *CalendarService) setupCalendarClients(vendor string, credentialsFile string) {
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		Logger.Sugar().Errorf("Unable to read client secret file: %v", credentialsFile)
		Logger.Sugar().Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		Logger.Sugar().Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config, vendor)
	c.vendorClientMap[vendor] = client

	ctx := context.Background()
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		Logger.Sugar().Fatalf("Unable to retrieve Calendar client: %v", err)

	}
	c.vendorServiceMap[vendor] = srv
}

func (c *CalendarService) loadVendorEvents(vendor string) {
	Logger.Sugar().Infof("Getting events for %s", vendor)
	srv := c.vendorServiceMap[vendor]

	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		Logger.Sugar().Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	if len(events.Items) == 0 {
		Logger.Sugar().Infof("No upcoming events found for %s", vendor)
	} else {
		c.filterEvents(events, vendor)
	}
}

func (c *CalendarService) filterEvents(events *calendar.Events, vendor string) {
	mapEventIds := map[string]bool{}

	var skipPrefix string
	if vendor == c.options.Destination {
		skipPrefix = fmt.Sprintf("[%s] ", c.options.Source)
	} else {
		skipPrefix = fmt.Sprintf("[%s] ", c.options.Destination)
	}

	for _, event := range events.Items {
		if strings.HasPrefix(event.Summary, skipPrefix) {
			continue
		}

		if event.RecurringEventId == "" {
			c.vendorEvents[vendor] = append(c.vendorEvents[vendor], event)
		} else {
			srv := c.vendorServiceMap[vendor]
			event, _ := srv.Events.Get("primary", event.RecurringEventId).Do()
			if event != nil && !mapEventIds[event.Id] {
				c.vendorEvents[vendor] = append(c.vendorEvents[vendor], event)
				mapEventIds[event.Id] = true
			}
		}
	}
}

func (c *CalendarService) importEvents(source string, destination string) {
	if len(c.vendorEvents[source]) == 0 {
		Logger.Sugar().Infof("No events to import from %s to %s", source, destination)
		return
	}

	Logger.Sugar().Infof("[%s] => [%s] Importing %d events ", source, destination, len(c.vendorEvents[source]))
	srv := c.vendorServiceMap[destination]
	destCalendar, err := srv.Calendars.Get("primary").Do()
	if err != nil {
		Logger.Sugar().Fatalf("Unable to retrieve calendar: %v", err)
	}
	destinationEmail := destCalendar.Id

	for _, event := range c.vendorEvents[source] {

		if c.options.SyncOptions.RedactedSummary {
			event.Summary = "Redacted"
		}
		event.Summary = fmt.Sprintf("[%s] %s", source, event.Summary)

		if c.options.SyncOptions.RedactedDescription {
			event.Description = "Redacted"
		}

		if c.options.SyncOptions.DisableReminders {
			event.Reminders = &calendar.EventReminders{
				Overrides:       []*calendar.EventReminder{},
				UseDefault:      false,
				ForceSendFields: []string{"UseDefault"},
			}
		}

		if c.options.SyncOptions.RedactedLocation {
			event.Location = "Redacted"
		}

		if c.options.SyncOptions.RedactedAttendees {
			event.Attendees = []*calendar.EventAttendee{}
		}

		if c.options.SyncOptions.RedactedAtachments {
			event.Attachments = []*calendar.EventAttachment{}
		}

		event.Organizer = &calendar.EventOrganizer{}

		event.ColorId = c.options.SyncOptions.EventColor

		if c.options.SyncOptions.RedactedOrganizer {
			event.Organizer = &calendar.EventOrganizer{
				DisplayName: source,
				Email:       destinationEmail,
			}
		}

		_, err := srv.Events.Import("primary", event).Do()
		if err != nil {
			Logger.Sugar().Fatalf("Unable to add event to calendar: %v", err)
		}
	}
}

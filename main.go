package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting calendar-notifier")
	log.Println("Loading environment variables")
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	log.Println("Loaded environment variables")
	log.Println("Getting events")

	events, err := getEvents()

	if err != nil {
		log.Fatalf("Failed to get events: %s", err)
	}

	// Iterate over the events and parse them
	for _, event := range events {
		// Skip empty or invalid events
		if event == "" {
			continue
		}

		// Parse the event details
		summary := extractValue(event, "SUMMARY")
		start := extractValue(event, "DTSTART")

		log.Printf("Processing event: %s: %s", summary, start)

		// If the event's start date of the event is today or is one week from now, send a notification
		now := time.Now()
		oneWeekFromNow := now.AddDate(0, 0, 7)
		startComponents := strings.Split(start, ":")
		if len(startComponents) < 2 {
			log.Printf("Failed to parse event start date: %s", start)
			continue
		}
		eventStartDateString := strings.Split(start, ":")[1]
		eventStart, err := time.Parse("20060102", eventStartDateString)
		if err != nil {
			log.Printf("Failed to parse event start date: %s", err)
			continue
		}

		if eventStart.Day() == now.Day() || eventStart.Day() == oneWeekFromNow.Day() {
			log.Println("Event is today or one week from now, sending notification")
			niceDate := eventStart.Format("02/01/2006")
			message := fmt.Sprintf("%s: %s", summary, niceDate)
			err := sendTelegramMessage(message)
			if err != nil {
				log.Printf("Failed to send telegram message: %s", err)
			}
			log.Println("Sent notification")
		}
	}
	log.Println("Program finished")
}

// Helper function to extract property value from a VCALENDAR event
func extractValue(event, property string) string {
	start := strings.Index(event, property)
	if start == -1 {
		return ""
	}
	start += len(property) + 1
	end := strings.Index(event[start:], "\n")
	if end == -1 {
		return ""
	}
	return event[start : start+end]
}

func sendTelegramMessage(message string) error {
	httpClient := &http.Client{}
	telegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatId := os.Getenv("TELEGRAM_CHAT_ID")
	telegramApiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s", telegramBotToken, telegramChatId, message)
	req, err := http.NewRequest("GET", telegramApiUrl, nil)
	if err != nil {
		return err
	}
	_, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	return nil
}

func getEvents() ([]string, error) {
	calDavServerUrl := os.Getenv("CALDAV_SERVER_URL")
	calDavServerUsername := os.Getenv("CALDAV_SERVER_USERNAME")
	calDavServerPassword := os.Getenv("CALDAV_SERVER_PASSWORD")
	//telegramChatId := os.Getenv("TELEGRAM_CHAT_ID")
	//calanderName := os.Getenv("CALENDAR_NAME")

	// Connect to the CalDAV server
	// Create an HTTP client
	client := &http.Client{}

	// Create a GET request to fetch events
	req, err := http.NewRequest("REPORT", calDavServerUrl, nil)
	if err != nil {
		return nil, err
	}

	// Set basic authentication headers
	req.SetBasicAuth(calDavServerUsername, calDavServerPassword)

	// Set CalDAV specific headers
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("Depth", "1")

	// Specify the CalDAV XML body for the request
	xmlBody := `<?xml version="1.0" encoding="utf-8" ?>
		<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
		  <D:prop>
			<D:getetag/>
			<C:calendar-data>
			  <C:comp name="VCALENDAR">
				<C:prop name="VERSION"/>
				<C:comp name="VEVENT">
				  <C:prop name="UID"/>
				  <C:prop name="SUMMARY"/>
				  <C:prop name="DTSTART"/>
				  <C:prop name="DTEND"/>
				</C:comp>
			  </C:comp>
			</C:calendar-data>
		  </D:prop>
		  <C:filter>
			<C:comp-filter name="VEVENT"/>
		  </C:filter>
		</C:calendar-query>`

	// Set the request body
	req.Body = ioutil.NopCloser(strings.NewReader(xmlBody))
	req.ContentLength = int64(len(xmlBody))

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	okStatusCodes := []int{200, 207}
	requestOk := false
	for _, statusCode := range okStatusCodes {
		if resp.StatusCode == statusCode {
			requestOk = true
			break
		}
	}

	if !requestOk {
		return nil, err
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the response body as string
	response := string(body)

	// Split the response into individual events (assuming VCALENDAR format)
	return strings.Split(response, "BEGIN:VEVENT"), nil
}

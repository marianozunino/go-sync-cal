package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/pkg/browser"

	"golang.org/x/oauth2"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, vendor string) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokenFileName := fmt.Sprintf("%s-token.json", vendor)
	tok, err := tokenFromFile(tokenFileName)

	if err != nil {
		tok = getTokenFromWeb(config, vendor)
		saveToken(tokenFileName, tok)
	}
	return config.Client(context.Background(), tok)

}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// // Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config, vendor string) *oauth2.Token {
	config.RedirectURL = "http://localhost:5000"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	Logger.Sugar().Infof("A browser window will open to authorize this application to access your %s account.", vendor)
	Logger.Sugar().Infof("If the browser window does not open, please visit the following URL:\n%s", authURL)

	if err := browser.OpenURL(authURL); err != nil {
		Logger.Sugar().Fatalf("Unable to open browser: %v", err)
	}

	var authCode string
	wg := sync.WaitGroup{}
	wg.Add(1)

	mux := http.NewServeMux()

	httpServer := &http.Server{
		Addr:    ":5000",
		Handler: mux,
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "You can close this window now.")
		Logger.Sugar().Infof("Received authorization code: %s", r.URL.Query().Get("code"))
		authCode = r.URL.Query().Get("code")
		if authCode == "" {
			Logger.Sugar().Fatalf("Unable to retrieve authorization code from web. Did you allow access?")
		}
		err := httpServer.Close()
		if err != nil {
			Logger.Sugar().Fatalf("Unable to shutdown server: %v", err)
		}
		wg.Done()
	})
	Logger.Sugar().Infof("Starting server on port :5000")

	go httpServer.ListenAndServe()

	wg.Wait()

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		Logger.Sugar().Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	Logger.Sugar().Infof("Saving credential file to: %s", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		Logger.Sugar().Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

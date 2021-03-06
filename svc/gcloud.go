package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/davidecavestro/gmail-exporter/logger"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

// Retrieve a token, saves the token, then returns the generated client.
func GetClient(config *oauth2.Config, TokenFile string, BatchMode bool, NoBrowser bool, NoTokenSave bool) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := tokenFromFile(TokenFile)
	if err != nil {
		if BatchMode {
			logger.Fatalf("Cannot retrieve a valid token from file: %v\n%v", TokenFile, err)
		}
		tok = getTokenFromWeb(config, NoBrowser)
		if !NoTokenSave {
			err = saveToken(TokenFile, tok)
			if err != nil {
				logger.Errorf("Cannot save token from file: %v\n%v", TokenFile, err)
			}
		}
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config, NoBrowser bool) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)
	if !NoBrowser {
		if err := browser.OpenURL(authURL); err != nil {
			logger.Errorf("Unable to open web browser: %v", err)
		}
	}
	fmt.Printf("Authorization code:\n")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		logger.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		logger.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
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

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	logger.Debugf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		return err
	}
	return nil
}

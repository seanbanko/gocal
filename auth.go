package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// ----------------------------------------------------------------------
// This code inspired by
// https://github.com/googleworkspace/go-samples/blob/main/calendar/quickstart/quickstart.go
// https://developers.google.com/youtube/v3/code_samples/go
// ----------------------------------------------------------------------

func newCalendarService() *calendar.Service {
	ctx := context.Background()
	f, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read GoCal credentials file: %v", err)
	}
	config, err := google.ConfigFromJSON(f, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("Unable to parse GoCal credentials file to config: %v", err)
	}
	config.RedirectURL = "http://localhost:8090"
	tokFilePath, err := tokenFilePath()
	if err != nil {
		log.Printf("Unable to generate path to token file: %v", err)
		log.Print("Trying standard path")
		tokFilePath = "~/.gocal/token.json"
	}
	tok, err := tokenFromFile(tokFilePath)
	if err != nil {
		log.Printf("Error getting token from filepath: %v", err)
		log.Print("Attempting to get a new token from the web")
		tok, err = getTokenFromWeb(config)
		if err != nil {
			log.Printf("Error getting token from the web: %v", err)
			log.Print("Prompting user to get the token manually")
			tok, err = getTokenFromPrompt(config)
			if err != nil {
				log.Fatalf("Error getting token from prompt: %v", err)
			}
		}
		err = saveToken(tokFilePath, tok)
		if err != nil {
			log.Printf("Error saving token: %v", err)
		}
	}
	client := config.Client(ctx, tok)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Google Calendar client: %v", err)
	}
	return srv
}

// startWebServer starts a web server that listens on http://localhost:8090.
// The webserver waits for an oauth code in the three-legged auth flow.
func startWebServer() (codeCh chan string, err error) {
	listener, err := net.Listen("tcp", "localhost:8090")
	if err != nil {
		return nil, err
	}
	codeCh = make(chan string)
	// TODO handle errors here
	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		codeCh <- code
		listener.Close()
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Authorization code received.\nYou can now close this browser window and return to your application.")
	}))
	return codeCh, nil
}

func openURL(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		err = fmt.Errorf("Failed to identify operating system to open the URL")
	}
	return err
}

func exchangeCodeForToken(config *oauth2.Config, code string) (*oauth2.Token, error) {
	tok, err := config.Exchange(context.TODO(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token %v", err)
	}
	return tok, nil
}

func getTokenFromPrompt(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Welcome to GoCal, a TUI for Google Calendar\n")
	fmt.Printf("To use GoCal, you'll need to authorize GoCal to access your Google Calendar\n")
	fmt.Printf("Go to the following link in your browser to authorize GoCal:\n\n%v\n\n", authURL)
	fmt.Printf("After you sign in, your will be redirected to a page containing the authorization code in the URL.\n")
	example := "http://localhost/?state=state-token&code=[--CODE HERE--]&scope=https://www.googleapis.com/auth/calendar.events"
	fmt.Printf("It will look like this:\n\n%v\n\n", example)
	fmt.Printf("Type or copy-and-paste that authorization code here, then press enter\n")
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}
	return exchangeCodeForToken(config, code)
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	codeCh, err := startWebServer()
	if err != nil {
		log.Printf("Unable to start a web server: %v", err)
		return nil, err
	}
	err = openURL(authURL)
	if err != nil {
		log.Printf("Unable to open authorization URL in web server: %v", err)
		return nil, err
	}
	fmt.Println("Your browser has been opened to the following URL so that you can sign in with Google.")
	fmt.Println("")
	fmt.Println(authURL)
	fmt.Println("")
	fmt.Println("GoCal will resume once you've signed in.")
	code := <-codeCh
	return exchangeCodeForToken(config, code)
}

func tokenFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".config", "gocal")
	err = os.MkdirAll(tokenCacheDir, 0o700)
	if err != nil {
		return "", err
	}
	return filepath.Join(tokenCacheDir, "token.json"), nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	return t, err
}

func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(token)
	return err
}

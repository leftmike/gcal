package oauth2gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	oauth2cli "github.com/adrianmartinmulesoft/oauth2-auth-cli"
)

func loadToken(tokenFile string) (*oauth2.Token, error) {
	f, err := os.Open(tokenFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var token oauth2.Token
	err = json.NewDecoder(f).Decode(&token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func saveToken(tokenFile string, token *oauth2.Token) error {
	f, err := os.OpenFile(tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("oauth2gcp: unable to save token: %s: %s", tokenFile, err)
	}
	defer f.Close()

	json.NewEncoder(f).Encode(token)
	return nil
}

func GetClient(ctx context.Context, dir string, scopes ...string) (*http.Client, error) {
	credFile := filepath.Join(dir, "credentials.json")
	b, err := os.ReadFile(credFile)
	if err != nil {
		return nil, fmt.Errorf("oauth2gcp: missing client credentials: %s", err)
	}

	config, err := google.ConfigFromJSON(b, scopes...)
	if err != nil {
		return nil, fmt.Errorf("oauth2gcp: failed to parse client credentials: %s: %s",
			credFile, err)
	}

	tokenFile := filepath.Join(dir, "token.json")
	token, err := loadToken(tokenFile)
	if err != nil {
		token, err = oauth2cli.Authorize(config)
		if err != nil {
			return nil, fmt.Errorf("oauth2gcp: unable to authorize: %s", err)
		}
		err = saveToken(tokenFile, token)
		if err != nil {
			return nil, fmt.Errorf("oauth2gcp: unable to save token: %s", err)
		}
	}

	return config.Client(ctx, token), nil
}

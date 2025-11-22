package fishaudio

import (
    "errors"
    "net/http"
    "os"
)

type Client struct {
    APIKey  string
    BaseURL string
    HTTP    *http.Client
}

func NewClient(apiKey string) (*Client, error) {
    if apiKey == "" {
        apiKey = os.Getenv("FISH_API_KEY")
    }
    if apiKey == "" {
        return nil, errors.New("missing API key")
    }
    return &Client{APIKey: apiKey, BaseURL: "https://api.fish.audio", HTTP: &http.Client{}}, nil
}
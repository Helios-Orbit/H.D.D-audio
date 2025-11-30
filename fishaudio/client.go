package fishaudio

import (
    "errors"
    "net/http"
    "os"
    "time"
)

type Client struct {
    APIKey  string
    BaseURL string
    HTTP    *http.Client
    Pool    *WSConnPool
    Options ClientOptions
}

func NewClient(apiKey string) (*Client, error) {
    if apiKey == "" {
        apiKey = os.Getenv("FISH_API_KEY")
    }
    if apiKey == "" {
        return nil, errors.New("missing API key")
    }
    c := &Client{APIKey: apiKey, BaseURL: "https://api.fish.audio", HTTP: &http.Client{}}
    c.Options = ClientOptions{DefaultPooling: true, MaxConnsPerKey: 4, IdleTTL: 60 * time.Second, MaxLife: 10 * time.Minute, WSReadTimeout: 30 * time.Second, WSPingInterval: 15 * time.Second, AudioBuf: 256, PacketsBuf: 1024, TextIdleTTL: 2 * time.Minute}
    c.Pool = NewWSConnPool(c.Options.MaxConnsPerKey, c.Options.IdleTTL, c.Options.MaxLife, c.Options.TextIdleTTL)
    return c, nil
}

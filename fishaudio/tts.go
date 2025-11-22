package fishaudio

import (
    "bytes"
    "context"
    "io"
    "net/http"
    "github.com/vmihailenco/msgpack/v5"
)

func (c *Client) Convert(ctx context.Context, req TTSRequest, backend string) (io.ReadCloser, int, error) {
    b, err := msgpack.Marshal(req)
    if err != nil {
        return nil, 0, err
    }
    u := c.BaseURL + "/v1/tts"
    r, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(b))
    if err != nil {
        return nil, 0, err
    }
    r.Header.Set("Authorization", "Bearer "+c.APIKey)
    r.Header.Set("model", backend)
    r.Header.Set("Content-Type", "application/msgpack")
    resp, err := c.HTTP.Do(r)
    if err != nil {
        return nil, 0, err
    }
    if resp.StatusCode >= 200 && resp.StatusCode < 400 {
        return resp.Body, resp.StatusCode, nil
    }
    defer resp.Body.Close()
    return nil, resp.StatusCode, io.EOF
}
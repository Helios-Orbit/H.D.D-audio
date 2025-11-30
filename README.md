# H.D.D Audio SDK

Fairy Audio Golang SDK, used for Helios Project

## Features
- Batch TTS over `HTTP POST /v1/tts` with MsgPack requests
- Realtime TTS over `WS /v1/tts/live` streaming audio chunks
- Simple client with `Authorization: Bearer <FISH_API_KEY>`
- Voice conditioning via `reference_id` and prosody controls (speed, volume)
- Flexible output: `mp3`, `opus`, `wav`, `pcm`, configurable sample rate and bitrates
- Low‑latency streaming pipeline with flush control
- Default WebSocket connection pooling by `BaseURL|backend|format|reference_id` with concurrent reuse (multi-conn)

## Requirements
- `Go 1.21`
- Fish Audio API key (`FISH_API_KEY`)
- Dependencies: `github.com/gorilla/websocket`, `github.com/vmihailenco/msgpack/v5`

## Install
This SDK’s module name is `fishaudio`. If you use it from another project, add a replace directive in your project’s `go.mod` to point to this repository:

```go
module your-module

go 1.21

require fishaudio v0.0.0

replace fishaudio => ../H.D.D-audio
```

Then import the package as `fishaudio/fishaudio`.

## Configuration
- `FISH_API_KEY`: required unless passed to `NewClient("...")`
- `FISH_REFERENCE_ID`: optional voice reference id to condition Fairy’s timbre
- Backend selection via the `model` header (e.g., `"s1"`)
 - Connection pool defaults: `MaxConnsPerKey=4`, `IdleTTL=60s`, `MaxLife=10m`
 - Buffers: `AudioBuf=256`, `PacketsBuf=1024`

## Usage

### Batch TTS (HTTP)
```go
package main

import (
    "context"
    "os"
    "io"
    fa "fishaudio/fishaudio"
)

func strPtr(s string) *string { return &s }

func main() {
    client, err := fa.NewClient(os.Getenv("FISH_API_KEY"))
    if err != nil { panic(err) }

    req := fa.TTSRequest{ Text: "Hello from Fairy" }
    req.Format = strPtr("mp3")

    body, _, err := client.Convert(context.Background(), req, "s1")
    if err != nil { panic(err) }
    defer body.Close()

    f, _ := os.Create("out.mp3")
    defer f.Close()
    _, _ = io.Copy(f, body)
}
```

### Realtime TTS (WebSocket)
```go
package main

import (
    "context"
    "os"
    "time"
    "io"
    fa "fishaudio/fishaudio"
)

func strPtr(s string) *string { return &s }

func main() {
    c, err := fa.NewClient(os.Getenv("FISH_API_KEY"))
    if err != nil { panic(err) }

    req := fa.TTSRequest{ Text: "" }
    req.Format = strPtr("mp3")

    texts := make(chan string, 2)
    go func() {
        texts <- "Welcome to Fish Audio realtime synthesis."
        texts <- "This is Fairy speaking in Helios."
        close(texts)
    }()

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    conn, err := c.ConvertRealtime(ctx, req, texts, "s1")
    if err != nil { panic(err) }

    out, _ := os.Create("out_rt.mp3")
    defer out.Close()

    for {
        select {
        case a := <-conn.Audio:
            _, _ = out.Write(a)
        case err := <-conn.Error:
            panic(err)
        case <-conn.Close:
            return
        case <-ctx.Done():
            return
        }
    }
}
```

### Pooling and lifecycle
- Pool key: `BaseURL|backend|format|reference_id`.
- `RealtimeConnection.Close()`: release the lease and keep WS open in pool.
- `RealtimeConnection.ForceClose()`: close WS and remove from pool.
- `RealtimeConnection.Done()`: session completion signal.

### Performance notes
- Struct-based MsgPack events with encoder reuse for lower allocations.
- Ogg/Opus demuxer with buffer limit and reset to prevent memory growth.

## API
- `client.go` (`fishaudio/client.go:15`): `NewClient(apiKey string) (*Client, error)`; reads `FISH_API_KEY` when empty; default `BaseURL=https://api.fish.audio`.
- `tts.go` (`fishaudio/tts.go:11`): `Convert(ctx, req, backend) (io.ReadCloser, status, error)`; POST MsgPack to `/v1/tts`.
- `realtime.go` (`fishaudio/realtime.go:21`): `ConvertRealtime(ctx, req, texts, backend) (*RealtimeConnection, error)`; WS `wss://api.fish.audio/v1/tts/live`; default pooled connection.
- `types.go` (`fishaudio/types.go:8`): `TTSRequest` with fields for text, prosody, format, sample rate, bitrates, latency, reference id.

## Security
- Keep `FISH_API_KEY` in secure storage or environment variables
- Do not commit secrets; never print the key

## License
MIT License. See `LICENSE`.
## Audio Formats
- WAV / PCM
  - Sample Rate: 8kHz, 16kHz, 24kHz, 32kHz, 44.1kHz
  - Default Sample Rate: 44.1kHz
  - 16-bit, mono
- MP3
  - Sample Rate: 32kHz, 44.1kHz (default)
  - Bitrate: 64kbps, 128kbps (default), 192kbps
  - mono
- Opus
  - Sample Rate: 48kHz (default)
  - Bitrate: -1000 (auto), 24kbps, 32kbps (default), 48kbps, 64kbps
  - mono

package main

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "os"
    "time"
    fa "fishaudio/fishaudio"
)

func strPtr(s string) *string { return &s }

func main() {
    key := os.Getenv("FISH_API_KEY")
    client, err := fa.NewClient(key)
    if err != nil { fmt.Println("err:", err); return }

    req := fa.TTSRequest{ Text: "这是一次 Opus 输出验证" }
    req.Format = strPtr("opus")
    if rid := os.Getenv("FISH_REFERENCE_ID"); rid != "" { req.ReferenceID = &rid }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    body, _, err := client.Convert(ctx, req, "s1")
    if err != nil { fmt.Println("http err:", err); return }
    defer body.Close()

    data, err := io.ReadAll(body)
    if err != nil { fmt.Println("read err:", err); return }
    if err := os.WriteFile("out_opus.opus", data, 0644); err != nil { fmt.Println("write err:", err); return }

    ok := bytes.Contains(data, []byte("OpusHead")) || bytes.HasPrefix(data, []byte("OggS"))
    if ok {
        fmt.Println("OK: detected Opus container signature; written out_opus.opus")
    } else {
        fmt.Println("WARN: no Opus/Ogg signature found; file written, please inspect with ffprobe")
    }
}
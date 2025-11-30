package main

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "os"
    "os/exec"
    "time"
    fa "fishaudio/fishaudio"
)

func strPtr(s string) *string { return &s }

func wavHeader(dataLen int, sampleRate int) []byte {
    b := &bytes.Buffer{}
    br := sampleRate * 2
    hl := 36 + dataLen
    dl := dataLen
    b.WriteString("RIFF")
    b.Write([]byte{byte(hl), byte(hl >> 8), byte(hl >> 16), byte(hl >> 24)})
    b.WriteString("WAVE")
    b.WriteString("fmt ")
    b.Write([]byte{16, 0, 0, 0})
    b.Write([]byte{1, 0})
    b.Write([]byte{1, 0})
    b.Write([]byte{byte(sampleRate), byte(sampleRate >> 8), byte(sampleRate >> 16), byte(sampleRate >> 24)})
    b.Write([]byte{byte(br), byte(br >> 8), byte(br >> 16), byte(br >> 24)})
    b.Write([]byte{2, 0})
    b.Write([]byte{16, 0})
    b.WriteString("data")
    b.Write([]byte{byte(dl), byte(dl >> 8), byte(dl >> 16), byte(dl >> 24)})
    return b.Bytes()
}

func main() {
    key := os.Getenv("FISH_API_KEY")
    c, err := fa.NewClient(key)
    if err != nil { fmt.Println("err:", err); return }
    format := os.Getenv("FISH_FORMAT")
    if format == "" { format = "wav" }
    req := fa.TTSRequest{ Text: "这是 WAV/PCM 验证" }
    req.Format = &format
    sr := 44100
    req.SampleRate = &sr
    if rid := os.Getenv("FISH_REFERENCE_ID"); rid != "" { req.ReferenceID = &rid }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    body, _, err := c.Convert(ctx, req, "s1")
    if err != nil { fmt.Println("http err:", err); return }
    defer body.Close()
    data, err := io.ReadAll(body)
    if err != nil { fmt.Println("read err:", err); return }
    if format == "wav" {
        _ = os.WriteFile("out_wav.wav", data, 0644)
        if path, err := exec.LookPath("ffplay"); err == nil {
            cmd := exec.Command(path, "-autoexit", "-nodisp", "-hide_banner", "-loglevel", "quiet", "-nostats", "out_wav.wav")
            cmd.Stdout = io.Discard
            cmd.Stderr = io.Discard
            _ = cmd.Run()
        }
        fmt.Println("written out_wav.wav")
    } else if format == "pcm" {
        _ = os.WriteFile("out_pcm.pcm", data, 0644)
        wh := wavHeader(len(data), sr)
        wf := append(wh, data...)
        _ = os.WriteFile("out_pcm.wav", wf, 0644)
        if path, err := exec.LookPath("ffplay"); err == nil {
            cmd := exec.Command(path, "-autoexit", "-nodisp", "-hide_banner", "-loglevel", "quiet", "-nostats", "-f", "s16le", "-ar", fmt.Sprintf("%d", sr), "-ac", "1", "out_pcm.pcm")
            cmd.Stdout = io.Discard
            cmd.Stderr = io.Discard
            _ = cmd.Run()
        }
        fmt.Println("written out_pcm.pcm and out_pcm.wav")
    } else {
        fmt.Println("unsupported FISH_FORMAT; set wav or pcm")
    }
}
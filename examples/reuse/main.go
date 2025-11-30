package main

import (
    "context"
    "fmt"
    "math/rand"
    "os"
    "os/exec"
    "io"
    "time"
    fa "fishaudio/fishaudio"
)

func strPtr(s string) *string { return &s }

func makeTexts(base time.Duration, jitterMax time.Duration, sentences []string) <-chan string {
    ch := make(chan string, len(sentences))
    go func() {
        r := rand.New(rand.NewSource(time.Now().UnixNano()))
        for _, s := range sentences {
            ch <- s
            j := time.Duration(r.Intn(int(jitterMax/time.Millisecond))) * time.Millisecond
            time.Sleep(base + j)
        }
        close(ch)
    }()
    return ch
}

func startPlayer() (stdin io.WriteCloser, stop func()) {
    if path, err := exec.LookPath("ffplay"); err == nil {
        cmd := exec.Command(path, "-autoexit", "-nodisp", "-hide_banner", "-loglevel", "quiet", "-nostats", "-")
        cmd.Stdout = io.Discard
        cmd.Stderr = io.Discard
        w, err := cmd.StdinPipe()
        if err == nil {
            _ = cmd.Start()
            return w, func() { _ = w.Close(); _ = cmd.Wait() }
        }
    }
    return nil, func() {}
}

func runTwoPhases(c *fa.Client, req fa.TTSRequest, backend, outfile string, a []string, b []string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
    defer cancel()
    texts := make(chan string, len(a)+len(b))
    dialStart := time.Now()
    conn, err := c.ConvertRealtime(ctx, req, texts, backend)
    if err != nil { return err }
    <-conn.Open
    fmt.Printf("connected in %dms\n", time.Since(dialStart).Milliseconds())
    out, err := os.Create(outfile)
    if err != nil { return err }
    defer out.Close()
    playerIn, stop := startPlayer()
    defer stop()
    go func() {
        r := rand.New(rand.NewSource(time.Now().UnixNano()))
        for _, s := range a { texts <- s; j := time.Duration(r.Intn(200)) * time.Millisecond; time.Sleep(300*time.Millisecond + j) }
        time.Sleep(500 * time.Millisecond)
        for _, s := range b { texts <- s; j := time.Duration(r.Intn(200)) * time.Millisecond; time.Sleep(300*time.Millisecond + j) }
        close(texts)
    }()
    for {
        select {
        case a := <-conn.Audio:
            if playerIn != nil { _, _ = playerIn.Write(a) }
            _, _ = out.Write(a)
        case e := <-conn.Error:
            conn.ForceClose()
            return e
        case <-conn.DoneCh():
            return nil
        case <-ctx.Done():
            conn.Stop()
            conn.Release()
            return ctx.Err()
        }
    }
}

func main() {
    key := os.Getenv("FISH_API_KEY")
    c, err := fa.NewClient(key)
    if err != nil { fmt.Println("err:", err); return }
    req := fa.TTSRequest{ Text: "" }
    format := "mp3"
    req.Format = &format
    if rid := os.Getenv("FISH_REFERENCE_ID"); rid != "" { req.ReferenceID = &rid }
    s1 := []string{"会话一：第一句", "会话一：第二句", "会话一：第三句", "会话一：第四句", "会话一：第五句"}
    s2 := []string{"会话二：第一句", "会话二：第二句", "会话二：第三句", "会话二：第四句", "会话二：第五句"}
    fmt.Println("session A+B on same connection start")
    if err := runTwoPhases(c, req, "s1", "out_reuse_ab."+format, s1, s2); err != nil { fmt.Println("session err:", err) }
    fmt.Println("session A+B end")
}
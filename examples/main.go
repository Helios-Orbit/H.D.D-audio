package main

import (
    "context"
    "fmt"
    "os"
    "time"
    fa "fishaudio/fishaudio"
    "os/signal"
    "syscall"
    "os/exec"
    "io"
)

func main() {
    key := os.Getenv("FISH_API_KEY")
    client, err := fa.NewClient(key)
    if err != nil { fmt.Println("err:", err); return }

    // Build realtime request: empty text, optional reference_id, mp3 format
    req := fa.TTSRequest{ Text: "" }
    format := "mp3"
    req.Format = &format
    if rid := os.Getenv("FISH_REFERENCE_ID"); rid != "" { req.ReferenceID = &rid }

    // Prepare text stream
    texts := make(chan string, 3)
    go func() {
        chunks := []string{
            "你好，欢迎使用 Fish Audio 实时语音合成服务，这里是新艾利都最强人工智能 Fairy 的 Golang SDK 测试。",
            "绳匠，我通过读取店内监控，发现一位顾客偷走了货架上的录像带。我已经把相关视频发给了治安局，并将此人列入了本店的「猎杀名单」。下次他再进入商店时，伊埃斯会冲上去对他使用上勾拳。",
            "对了，亚马逊和京东上不少商品正在打折，您可以购买内存条提升我的运算，或购买高清摄像头加强我的扫描能力。当然，您也可以什么都不买，我是不会有任何怨言的，毕竟我只是个AI。",
            "Fairy 天气小助手提醒您，今天部分空洞区域会有降雨。好消息是，以骸讨厌雨。坏消息是，以骸更讨厌您。",
        }
        for _, c := range chunks { texts <- c; time.Sleep(200 * time.Millisecond) }
        close(texts)
    }()

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    conn, err := client.ConvertRealtime(ctx, req, texts, "s1")
    if err != nil { fmt.Println("realtime err:", err); return }

    out, err := os.Create("out_rt.mp3")
    if err != nil { fmt.Println("file err:", err); return }
    defer out.Close()

    var ffplayStdin io.WriteCloser
    var ffplayCmd *exec.Cmd
    if path, err := exec.LookPath("ffplay"); err == nil {
        ffplayCmd = exec.Command(path, "-autoexit", "-", "-nodisp", "-hide_banner", "-loglevel", "quiet", "-nostats")
        ffplayCmd.Stdout = io.Discard
        ffplayCmd.Stderr = io.Discard
        ffplayStdin, err = ffplayCmd.StdinPipe()
        if err == nil { _ = ffplayCmd.Start() } else { ffplayCmd = nil }
    }

    // Handle Ctrl+C to close
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

    opened := false
    for {
        select {
        case <-conn.Open:
            opened = true
            fmt.Println("ws open")
        case a := <-conn.Audio:
            if _, err := out.Write(a); err != nil { fmt.Println("write err:", err) }
            if ffplayStdin != nil { _, _ = ffplayStdin.Write(a) }
        case err := <-conn.Error:
            fmt.Println("ws error:", err)
            return
        case <-conn.Close:
            fmt.Println("ws close")
            if ffplayStdin != nil { _ = ffplayStdin.Close() }
            if ffplayCmd != nil { _ = ffplayCmd.Wait() }
            if opened { fmt.Println("written out_rt.mp3") }
            if ffplayCmd == nil {
                if path, err := exec.LookPath("afplay"); err == nil {
                    cmd := exec.Command(path, "out_rt.mp3")
                    cmd.Stdout = io.Discard
                    cmd.Stderr = io.Discard
                    _ = cmd.Run()
                }
            }
            return
        case <-sig:
            fmt.Println("signal received, exit")
            return
        case <-ctx.Done():
            fmt.Println("timeout")
            return
        }
    }
}
# H.D.D Audio SDK（中文）

Fairy Audio Golang SDK，用于 Helios 项目

## 功能特性
- 批量 TTS：通过 `HTTP POST /v1/tts`，请求体使用 MsgPack
- 实时 TTS：通过 `WS /v1/tts/live`，流式返回音频片段
- 简单客户端：`Authorization: Bearer <FISH_API_KEY>`
- 声音条件：支持 `reference_id` 与韵律参数（速度、音量）
- 灵活输出：`mp3`、`opus`、`wav`、`pcm`，可配置采样率与码率
- 低延迟：支持 `flush` 控制的流式管线
- 默认启用 WebSocket 连接池：按 `BaseURL|backend|format|reference_id` 池化，支持并发复用（多连接）

## 环境要求
- `Go 1.21`
- Fish Audio API key（`FISH_API_KEY`）
- 依赖：`github.com/gorilla/websocket`、`github.com/vmihailenco/msgpack/v5`

## 安装
本 SDK 的模块名为 `fishaudio`。如从其他项目使用，可在该项目的 `go.mod` 中添加 replace 指令指向本仓库：

```go
module your-module

go 1.21

require fishaudio v0.0.0

replace fishaudio => ../H.D.D-audio
```

然后以 `fishaudio/fishaudio` 方式导入包。

## 配置
- `FISH_API_KEY`：必填；也可在调用 `NewClient("...")` 传入
- `FISH_REFERENCE_ID`：可选，用于音色条件（timbre）
- 后端选择：通过 `model` 头（例如 `"s1"`）
- 连接池默认：`MaxConnsPerKey=4`、`IdleTTL=60s`、`MaxLife=10m`
- 缓冲区：`AudioBuf=256`、`PacketsBuf=1024`

## 使用示例

### 批量 TTS（HTTP）
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

### 实时 TTS（WebSocket）
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

### 连接池与生命周期
- 池 key：`BaseURL|backend|format|reference_id`
- `RealtimeConnection.Close()`：释放租约，连接在池中保持打开以便复用
- `RealtimeConnection.ForceClose()`：强制关闭并从池移除
- `RealtimeConnection.Done()`：会话完成信号

### 性能说明
- 事件使用结构体并复用 MsgPack 编/解码器，降低分配与反射
- Ogg/Opus Demux 提供缓冲上限与 `Reset()`，避免异常流导致内存增长

## API
- `client.go`（`fishaudio/client.go:15`）：`NewClient(apiKey string) (*Client, error)`；当入参为空从环境读取 `FISH_API_KEY`；默认 `BaseURL=https://api.fish.audio`
- `tts.go`（`fishaudio/tts.go:11`）：`Convert(ctx, req, backend) (io.ReadCloser, status, error)`；向 `/v1/tts` 发送 MsgPack 请求
- `realtime.go`（`fishaudio/realtime.go:21`）：`ConvertRealtime(ctx, req, texts, backend) (*RealtimeConnection, error)`；`wss://api.fish.audio/v1/tts/live`；默认走连接池
- `types.go`（`fishaudio/types.go:8`）：`TTSRequest` 字段覆盖文本、韵律、格式、采样率、码率、延迟、reference id

## 安全
- 将 `FISH_API_KEY` 保存在安全存储或环境变量中
- 不要提交任何密钥；不要打印密钥

## 许可
MIT License，详见 `LICENSE`
## 音频格式
- WAV / PCM
  - 采样率：8kHz、16kHz、24kHz、32kHz、44.1kHz
  - 默认采样率：44.1kHz
  - 16-bit，单声道
- MP3
  - 采样率：32kHz、44.1kHz（默认）
  - 码率：64kbps、128kbps（默认）、192kbps
  - 单声道
- Opus
  - 采样率：48kHz（默认）
  - 码率：-1000（自动）、24kbps、32kbps（默认）、48kbps、64kbps
  - 单声道
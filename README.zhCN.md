# H.D.D Audio 
Fairy Audio Golang SDK, used for Helios Project

## 特性
- 基于 `HTTP POST /v1/tts` 的批量 TTS（MsgPack 请求体）
- 基于 `WS /v1/tts/live` 的实时 TTS 流式音频
- 简单易用的客户端，使用 `Authorization: Bearer <FISH_API_KEY>` 授权
- 支持通过 `reference_id` 与韵律（速度、音量）进行音色与风格调控
- 灵活输出：`mp3`、`opus`，可配置采样率与码率
- 低延迟流式管线，支持逐段 `flush`

## 环境要求
- `Go 1.21`
- Fish Audio API 密钥（`FISH_API_KEY`）
- 依赖：`github.com/gorilla/websocket`、`github.com/vmihailenco/msgpack/v5`

## 安装
本 SDK 的模块名为 `fishaudio`。若从其他项目中使用，请在你的项目 `go.mod` 添加 `replace` 指令指向本仓库所在路径：

```go
module your-module

go 1.21

require fishaudio v0.0.0

replace fishaudio => ../H.D.D-audio
```

随后以 `fishaudio/fishaudio` 进行导入。

## 配置
- `FISH_API_KEY`：必需，若未在 `NewClient("...")` 传入会从环境变量读取
- `FISH_REFERENCE_ID`：可选，用于音色参考与拟合
- 模型选择通过 `model` 请求头（例如 `"s1"`）

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

    req := fa.TTSRequest{ Text: "Fairy，您好" }
    req.Format = strPtr("mp3")

    body, status, err := client.Convert(context.Background(), req, "s1")
    if err != nil { panic(err) }
    defer body.Close()

    f, _ := os.Create("out.mp3")
    defer f.Close()
    _, _ = io.Copy(f, body)
    _ = status // 需要时可检查 HTTP 状态码
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
        texts <- "欢迎使用 Fish Audio 实时合成。"
        texts <- "这里是 Helios 项目的 Fairy。"
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

## API 速览
- `client.go`（`fishaudio/client.go:15`）：`NewClient(apiKey string)`；空字符串时读取 `FISH_API_KEY`；默认 `BaseURL=https://api.fish.audio`。
- `tts.go`（`fishaudio/tts.go:11`）：`Convert(ctx, req, backend)`；向 `/v1/tts` 发送 MsgPack 编码请求。
- `realtime.go`（`fishaudio/realtime.go:21`）：`ConvertRealtime(ctx, req, texts, backend)`；连接 `wss://api.fish.audio/v1/tts/live` 进行流式合成。
- `types.go`（`fishaudio/types.go:8`）：`TTSRequest` 定义文本、韵律、格式、采样率、码率、延迟、参考音色等字段。

## 安全建议
- 将 `FISH_API_KEY` 保存在安全的环境变量或密钥管理中
- 切勿将密钥写入代码库或日志

## 许可协议
MIT License，详见 `LICENSE`。
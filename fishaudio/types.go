package fishaudio

type Prosody struct {
    Speed  *float32 `msgpack:"speed,omitempty"`
    Volume *float32 `msgpack:"volume,omitempty"`
}

type TTSRequest struct {
    Text         string   `msgpack:"text"`
    Temperature  *float32 `msgpack:"temperature,omitempty"`
    TopP         *float32 `msgpack:"top_p,omitempty"`
    ReferenceID  *string  `msgpack:"reference_id,omitempty"`
    Prosody      *Prosody `msgpack:"prosody,omitempty"`
    ChunkLength  *int     `msgpack:"chunk_length,omitempty"`
    Normalize    *bool    `msgpack:"normalize,omitempty"`
    Format       *string  `msgpack:"format,omitempty"`
    SampleRate   *int     `msgpack:"sample_rate,omitempty"`
    Mp3Bitrate   *int     `msgpack:"mp3_bitrate,omitempty"`
    OpusBitrate  *int     `msgpack:"opus_bitrate,omitempty"`
    Latency      *string  `msgpack:"latency,omitempty"`
}
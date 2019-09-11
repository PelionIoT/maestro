package alerts

type Alert struct {
	Key string `json:"key"`
	Level string `json:"level"`
	Timestamp uint64 `json:"timestamp"`
	Metadata interface{} `json:"metadata"`
	Status bool `json:"status"`
}
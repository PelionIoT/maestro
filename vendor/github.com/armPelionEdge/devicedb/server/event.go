package server

import (
	"github.com/armPelionEdge/devicedb/historian"
	"encoding/json"
)

type event struct {
	Device         string      `json:"device"`
	Event          string      `json:"event"`
	Metadata       interface{} `json:"metadata"`
	Timestamp      uint64      `json:"timestamp"`
}

func MakeeventsFromEvents(es []*historian.Event) []*event {
	var events []*event = make([]*event, len(es))

	for i, e := range es {
		var metadata interface{}

		if err := json.Unmarshal([]byte(e.Data), &metadata); err != nil {
			metadata = e.Data
		}

		events[i] = &event{
			Device: e.SourceID,
			Event: e.Type,
			Metadata: metadata,
			Timestamp: e.Timestamp,
		}
	}

	return events
}
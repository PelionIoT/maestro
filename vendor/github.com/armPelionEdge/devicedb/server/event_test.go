package server_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/armPelionEdge/devicedb/historian"
	. "github.com/armPelionEdge/devicedb/server"
)

var _ = Describe("Event", func() {
	Describe("MakeeventsFromEvents", func() {
		It("Should decode json encoded metadata", func() {
			var events []*historian.Event = []*historian.Event{
				&historian.Event{ Data: "{}" },
				&historian.Event{ Data: "abc" },
				&historian.Event{ Data: "\"hello\"" },
			}

			encodedEvents := MakeeventsFromEvents(events)

			Expect(len(encodedEvents)).Should(Equal(3))
			Expect(encodedEvents[0].Metadata).Should(Equal(map[string]interface{}{ }))
			Expect(encodedEvents[1].Metadata).Should(Equal("abc"))
			Expect(encodedEvents[2].Metadata).Should(Equal("hello"))
		})
	})
})

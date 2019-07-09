package historian_test

import (
    "fmt"
    
    . "github.com/armPelionEdge/devicedb/historian"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/util"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Historian", func() {
    var (
        storageEngine StorageDriver
        historian *Historian
    )
    
    BeforeEach(func() {
        storageEngine = MakeNewStorageDriver()
        storageEngine.Open()
        
        historian = NewHistorian(storageEngine, 101, 0, 1000)
    })
    
    AfterEach(func() {
        storageEngine.Close()
    })
    
    Context("There are 100 logged events of 4 different varieties in the history from three different sources", func() {
        BeforeEach(func() {
            for i := 0; i < 100; i += 1 {
                historian.LogEvent(&Event{
                    Timestamp: uint64(i),
                    SourceID: fmt.Sprintf("source-%d", (i % 3)),
                    Type: fmt.Sprintf("type-%d", (i % 4)),
                    Data: fmt.Sprintf("data-%d", i >> 4),
                })
            }
        })
    
        Describe("performing a query filtering only by time range", func() {
            It("should return only events that happened between [50, 100) after the purge", func() {
                Expect(historian.Purge(&HistoryQuery{ Before: 50 })).Should(BeNil())
                
                iter, err := historian.Query(&HistoryQuery{ })
                
                Expect(err).Should(BeNil())
                Expect(historian.LogSize()).Should(Equal(uint64(50)))
                Expect(historian.LogSerial()).Should(Equal(uint64(101)))
                
                for i := 50; i < 100; i += 1 {
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
            
            It("should return only events that happened between [50, 100)", func() {
                var minSerial uint64 = 50
                iter, err := historian.Query(&HistoryQuery{ MinSerial: &minSerial })
                
                Expect(err).Should(BeNil())
                Expect(historian.LogSize()).Should(Equal(uint64(100)))
                Expect(historian.LogSerial()).Should(Equal(uint64(101)))
                
                for i := 49; i < 100; i += 1 {
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
            
            It("should return ten events from least to most recent when a time range between [0, 10) is queried and an ascending order is specified implictly", func() {
                iter, err := historian.Query(&HistoryQuery{
                    After: 0,
                    Before: 10,
                })
                
                Expect(err).Should(BeNil())
                
                for i := 0; i < 10; i += 1 {
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
            
            It("should return ten events from least to most recent when a time range between [0, 10) is queried and an ascending order is specified explicitly", func() {
                iter, err := historian.Query(&HistoryQuery{
                    After: 0,
                    Before: 10,
                    Order: "asc",
                })
                
                Expect(err).Should(BeNil())
                
                for i := 0; i < 10; i += 1 {
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
            
            It("should return ten events from most to least recent when a time range between [0, 10) is queried and a descending order is specified", func() {
                iter, err := historian.Query(&HistoryQuery{
                    After: 0,
                    Before: 10,
                    Order: "desc",
                })
                
                Expect(err).Should(BeNil())
                
                for i := 9; i >= 0; i -= 1 {
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
        })
        
        Describe("performing a query filtering only by source", func() {
            It("should return all events from source-0 in ascending order", func() {
                iter, err := historian.Query(&HistoryQuery{
                    Sources: []string{ "source-0" },
                })
                
                Expect(err).Should(BeNil())
                
                for i := 0; i < 100; i += 1 {
                    if i % 3 != 0 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
            
            It("should return all events from source-0 in ascending order", func() {
                iter, err := historian.Query(&HistoryQuery{
                    Sources: []string{ "source-0" },
                    Order: "asc",
                })
                
                Expect(err).Should(BeNil())
                
                for i := 0; i < 100; i += 1 {
                    if i % 3 != 0 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
            
            It("should return all events from source-0 in descending order", func() {
                iter, err := historian.Query(&HistoryQuery{
                    Sources: []string{ "source-0" },
                    Order: "desc",
                })
                
                Expect(err).Should(BeNil())
                
                for i := 99; i >= 0; i -= 1 {
                    if i % 3 != 0 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
            
            It("should return all events from source-0 and source-2 in ascending order", func() {
                iter, err := historian.Query(&HistoryQuery{
                    Sources: []string{ "source-0", "source-2" },
                })
                
                Expect(err).Should(BeNil())
                
                for i := 0; i < 100; i += 1 {
                    if i % 3 != 0 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                for i := 0; i < 100; i += 1 {
                    if i % 3 != 2 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
            
            It("should return all events from source-0 and source-2 in ascending order", func() {
                iter, err := historian.Query(&HistoryQuery{
                    Sources: []string{ "source-2", "source-0" },
                })
                
                Expect(err).Should(BeNil())
                
                for i := 0; i < 100; i += 1 {
                    if i % 3 != 0 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                for i := 0; i < 100; i += 1 {
                    if i % 3 != 2 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
        })
        
        Describe("performing a query filtering by source and time", func() {
            It("should return all events from source-0 and source-2 in ascending order between times [0, 50)", func() {
                iter, err := historian.Query(&HistoryQuery{
                    Sources: []string{ "source-0", "source-2" },
                    After: 0,
                    Before: 50,
                })
                
                Expect(err).Should(BeNil())
                
                for i := 0; i < 50; i += 1 {
                    if i % 3 != 0 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                for i := 0; i < 50; i += 1 {
                    if i % 3 != 2 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
        })
        
        Describe("performing a query filtering by source, time and data", func() {
            It("should return all events from source-0 and source-2 in ascending order between times [0, 50) whose data is data-0", func() {
                var data = "data-0"
                
                iter, err := historian.Query(&HistoryQuery{
                    Sources: []string{ "source-0", "source-2" },
                    After: 0,
                    Before: 50,
                    Data: &data,
                })
                
                Expect(err).Should(BeNil())
                
                for i := 0; i < 50; i += 1 {
                    if i % 3 != 0 || i >> 4 != 0 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                for i := 0; i < 50; i += 1 {
                    if i % 3 != 2 || i >> 4 != 0 {
                        continue
                    }
                    
                    Expect(iter.Next()).Should(BeTrue())
                    Expect(iter.Error()).Should(BeNil())
                    Expect(iter.Event()).Should(Not(BeNil()))
                    Expect(iter.Event().Timestamp).Should(Equal(uint64(i)))
                    Expect(iter.Event().SourceID).Should(Equal(fmt.Sprintf("source-%d", (i % 3))))
                    Expect(iter.Event().Type).Should(Equal(fmt.Sprintf("type-%d", (i % 4))))
                    Expect(iter.Event().Data).Should(Equal(fmt.Sprintf("data-%d", (i >> 4))))
                }
                
                Expect(iter.Next()).Should(BeFalse())
                Expect(iter.Error()).Should(BeNil())
                Expect(iter.Event()).Should(BeNil())
            })
        })
    })
})

package alerts_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/armPelionEdge/devicedb/alerts"
	. "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/util"
)

var _ = Describe("AlertStore", func() {
	var (
        storageEngine StorageDriver
        alertStore *AlertStoreImpl
    )
    
    BeforeEach(func() {
        storageEngine = MakeNewStorageDriver()
        storageEngine.Open()
        
        alertStore = NewAlertStore(storageEngine)
    })
    
    AfterEach(func() {
        storageEngine.Close()
	})
	
	It("Should work", func() {
		Expect(alertStore.Put(Alert{ Key: "abc" })).Should(BeNil())
		Expect(alertStore.Put(Alert{ Key: "def" })).Should(BeNil())
		Expect(alertStore.Put(Alert{ Key: "ghi" })).Should(BeNil())

		var alerts map[string]Alert = make(map[string]Alert)

		alertStore.ForEach(func(alert Alert) {
			alerts[alert.Key] = alert
		})

		Expect(alerts).Should(Equal(map[string]Alert{
			"abc": Alert{ Key: "abc" },
			"def": Alert{ Key: "def" },
			"ghi": Alert{ Key: "ghi" },
		}))

		Expect(alertStore.DeleteAll(map[string]Alert{ "abc": Alert{ Key: "abc" } })).Should(BeNil())

		alerts = make(map[string]Alert)

		alertStore.ForEach(func(alert Alert) {
			alerts[alert.Key] = alert
		})

		Expect(alerts).Should(Equal(map[string]Alert{
			"def": Alert{ Key: "def" },
			"ghi": Alert{ Key: "ghi" },
		}))
	})
})

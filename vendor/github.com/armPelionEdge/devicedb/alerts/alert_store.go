package alerts

import (
	"encoding/json"
	"github.com/armPelionEdge/devicedb/storage"
)

type AlertStoreImpl struct {
	storageDriver storage.StorageDriver
}

func NewAlertStore(storageDriver storage.StorageDriver) *AlertStoreImpl {
	return &AlertStoreImpl{
		storageDriver: storageDriver,
	}
}

func (alertStore *AlertStoreImpl) Put(alert Alert) error {
	encodedAlert, err := json.Marshal(alert)

	if err != nil {
		return err
	}

	batch := storage.NewBatch()
	batch.Put([]byte(alert.Key), encodedAlert)

	return alertStore.storageDriver.Batch(batch)
}

func (alertStore *AlertStoreImpl) DeleteAll(alerts map[string]Alert) error {
	batch := storage.NewBatch()

	for _, alert := range alerts {
		batch.Delete([]byte(alert.Key))
	}

	return alertStore.storageDriver.Batch(batch)
}

func (alertStore *AlertStoreImpl) ForEach(cb func(alert Alert)) error {
	iter, err := alertStore.storageDriver.GetMatches([][]byte{ []byte{ } })

	if err != nil {
		return err
	}

	defer iter.Release()

	for iter.Next() {
		var alert Alert

		if err := json.Unmarshal(iter.Value(), &alert); err != nil {
			return err
		}

		cb(alert)
	}

	return iter.Error()
}
package alerts_test

import (
	. "github.com/armPelionEdge/devicedb/alerts"
)

type MockAlertStore struct {
	alerts map[string]Alert
	putError error
	deleteAllError error
	forEachError error
}

func NewMockAlertStore() *MockAlertStore {
	return &MockAlertStore{
		alerts: make(map[string]Alert),
	}
}

func (alertStore *MockAlertStore) Put(alert Alert) error {
	if alertStore.putError != nil {
		return alertStore.putError
	}

	alertStore.alerts[alert.Key] = alert

	return nil
}

func (alertStore *MockAlertStore) Has(key string) bool {
	_, ok := alertStore.alerts[key]

	return ok
}

func (alertStore *MockAlertStore) Get(key string) Alert {
	return alertStore.alerts[key]
}

func (alertStore *MockAlertStore) DeleteAll(alerts map[string]Alert) error {
	if alertStore.deleteAllError != nil {
		return alertStore.deleteAllError
	}

	for _, a := range alerts {
		delete(alertStore.alerts, a.Key)
	}

	return nil
}

func (alertStore *MockAlertStore) ForEach(cb func(alert Alert)) error {
	if alertStore.forEachError != nil {
		return alertStore.forEachError
	}

	for _, a := range alertStore.alerts {
		cb(a)
	}

	return nil
}

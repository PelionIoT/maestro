package alerts

import (
	"sync"
)

type AlertStore interface {
	Put(alert Alert) error
	DeleteAll(alerts map[string]Alert) error
	ForEach(func(alert Alert)) error
}

type AlertMap struct {
	mu sync.Mutex
	alertStore AlertStore
}

func NewAlertMap(alertStore AlertStore) *AlertMap {
	return &AlertMap{
		alertStore: alertStore,
	}
}

func (alertMap *AlertMap) UpdateAlert(alert Alert) error {
	alertMap.mu.Lock()
	defer alertMap.mu.Unlock()

	return alertMap.alertStore.Put(alert)
}

func (alertMap *AlertMap) GetAlerts() (map[string]Alert, error) {
	var alerts map[string]Alert = make(map[string]Alert)

	err := alertMap.alertStore.ForEach(func(alert Alert) {
		alerts[alert.Key] = alert
	})

	if err != nil {
		return nil, err
	}

	return alerts, nil
}

// Blocks calls to UpdateAlert()
func (alertMap *AlertMap) ClearAlerts(alerts map[string]Alert) error {
	alertMap.mu.Lock()
	defer alertMap.mu.Unlock()

	var deleteAlerts map[string]Alert = make(map[string]Alert, len(alerts))

	for _, a := range alerts {
		deleteAlerts[a.Key] = a
	}

	err := alertMap.alertStore.ForEach(func(alert Alert) {
		if a, ok := alerts[alert.Key]; ok && alert.Timestamp != a.Timestamp {
			// This shouldn't be deleted since its value was changed since
			// reading. The new value will need to be forwarded later
			delete(deleteAlerts, a.Key)
		}
	})

	if err != nil {
		return err
	}

	return alertMap.alertStore.DeleteAll(deleteAlerts)
}
package storage

import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    prometheusStorageErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "devicedb_storage_errors",
		Help: "Counts the number of errors encountered in the disk storage layer",
    }, []string{
		"operation",
		"path",
    })
)

func init() {
    prometheus.MustRegister(prometheusStorageErrors)
}

func prometheusRecordStorageError(operation, path string) {
	prometheusStorageErrors.With(prometheus.Labels{
		"operation": operation,
		"path": path,
	}).Inc()
}
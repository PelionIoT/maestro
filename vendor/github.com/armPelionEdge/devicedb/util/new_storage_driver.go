package util

import (
    . "github.com/PelionIoT/devicedb/storage"
)

func MakeNewStorageDriver() StorageDriver {
    return NewLevelDBStorageDriver("/tmp/testdb-" + RandomString(), nil)
}
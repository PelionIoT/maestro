package util

import (
    . "github.com/armPelionEdge/devicedb/storage"
)

func MakeNewStorageDriver() StorageDriver {
    return NewLevelDBStorageDriver("/tmp/testdb-" + RandomString(), nil)
}
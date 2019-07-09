package partition

import (
    . "github.com/armPelionEdge/devicedb/data"
)

type PartitionIterator interface {
    Next() bool
    // The site that the current entry belongs to
    Site() string
    // The bucket that the current entry belongs to within its site
    Bucket() string
    // The key of the current entry
    Key() string
    // The value of the current entry
    Value() *SiblingSet
    // The checksum of the current entry
    Release()
    Error() error
}
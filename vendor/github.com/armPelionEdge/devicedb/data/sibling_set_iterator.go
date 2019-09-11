package data

type SiblingSetIterator interface {
    Next() bool
    Prefix() []byte
    Key() []byte
    Value() *SiblingSet
    LocalVersion() uint64
    Release()
    Error() error
}
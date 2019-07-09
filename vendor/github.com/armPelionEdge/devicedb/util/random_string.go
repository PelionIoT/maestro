package util

import (
    "fmt"
    "encoding/binary"
    "crypto/rand"
)

func RandomString() string {
    randomBytes := make([]byte, 16)
    rand.Read(randomBytes)
    
    high := binary.BigEndian.Uint64(randomBytes[:8])
    low := binary.BigEndian.Uint64(randomBytes[8:])
    
    return fmt.Sprintf("%05x%05x", high, low)
}
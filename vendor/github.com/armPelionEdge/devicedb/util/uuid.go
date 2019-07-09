package util

import (
    "encoding/binary"
    "crypto/rand"
    "github.com/google/uuid"
)

func UUID64() uint64 {
    randomBytes := make([]byte, 8)
    rand.Read(randomBytes)
    
    return binary.BigEndian.Uint64(randomBytes[:8])
}

func UUID() (string, error) {
    newUUID, err := uuid.NewRandom()

    if err != nil {
        return "", err
    }

    return newUUID.String(), nil
}
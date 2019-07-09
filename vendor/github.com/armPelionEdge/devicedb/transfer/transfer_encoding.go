package transfer

import (
    "io"
    "encoding/json"
)

type PartitionTransferEncoder interface {
    Encode() (io.Reader, error)
}

type PartitionTransferDecoder interface {
    Decode() (PartitionTransfer, error)
}

type TransferEncoder struct {
    transfer PartitionTransfer
    reader io.Reader
}

func NewTransferEncoder(transfer PartitionTransfer) *TransferEncoder {
    return &TransferEncoder{
        transfer: transfer,
    }
}

func (encoder *TransferEncoder) Encode() (io.Reader, error) {
    if encoder.reader != nil {
        return encoder.reader, nil
    }

    encoder.reader = &JSONPartitionReader{
        PartitionTransfer: encoder.transfer,
    }

    return encoder.reader, nil
}

type JSONPartitionReader struct {
    PartitionTransfer PartitionTransfer
    needsDelimiter bool
    currentChunk []byte
}

func (partitionReader *JSONPartitionReader) Read(p []byte) (n int, err error) {
    for len(p) > 0 {
        if len(partitionReader.currentChunk) == 0 {
            chunk, err := partitionReader.nextChunk()

            if err != nil {
                return n, err
            }

            if chunk == nil {
                return n, io.EOF
            }

            partitionReader.currentChunk = chunk
        }

        if partitionReader.needsDelimiter {
            nCopied := copy(p, []byte("\n"))
            p = p[nCopied:]
            n += nCopied
            partitionReader.needsDelimiter = false
        }

        nCopied := copy(p, partitionReader.currentChunk)
        p = p[nCopied:]
        n += nCopied
        partitionReader.currentChunk = partitionReader.currentChunk[nCopied:]

        // if we have copied a whole chunk the next character needs to be a delimeter
        partitionReader.needsDelimiter = len(partitionReader.currentChunk) == 0
    }

    return n, nil
}

func (partitionReader *JSONPartitionReader) nextChunk() ([]byte, error) {
    nextChunk, err := partitionReader.PartitionTransfer.NextChunk()

    if err != nil {
        return nil, err
    }

    if nextChunk.IsEmpty() {
        return nil, nil
    }

    return json.Marshal(nextChunk)
}

type TransferDecoder struct {
    transfer PartitionTransfer
}

func NewTransferDecoder(reader io.Reader) *TransferDecoder {
    return &TransferDecoder{
        transfer: NewIncomingTransfer(reader),
    }
}

func (decoder *TransferDecoder) Decode() (PartitionTransfer, error) {
    return decoder.transfer, nil
}
package data

import (
    "encoding/json"
)

type Row struct {
    Key string `json:"key"`
    LocalVersion uint64 `json:"localVersion"`
    Siblings *SiblingSet `json:"siblings"`
}

func (row *Row) Encode() []byte {
    result, _ := json.Marshal(row)

    return result
}

func (row *Row) Decode(encodedRow []byte, formatVersion string) error {
    if formatVersion == "0" {
        var siblingSet SiblingSet

        err := siblingSet.Decode(encodedRow)

        if err == nil {
            row.LocalVersion = 0
            row.Siblings = &siblingSet

            return nil
        }

        // if it fails to decode using format vesion
        // 0 which was without the row type then it might
        // have been converted already to the new format
        // version in a partially completed upgrade before.
        // in this case, try to decode it as a regular row
        // before returning an error
    }

    return json.Unmarshal(encodedRow, row)
}
package processor

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

var ErrData = errors.New("data error")

type DataVersion int

const (
	DataVersion_Undefined = iota
	DataVersion_1
	DataVersion_2
)

type Scan struct {
	IP          string      `json:"ip"`
	Port        uint32      `json:"port"`
	Service     string      `json:"service"`
	Timestamp   int64       `json:"timestamp"`
	DataVersion DataVersion `json:"data_version"`
	Data        struct {
		V1Data
		V2Data
	} `json:"data"`
}

type V1Data struct {
	ResponseBytesUtf8 []byte `json:"response_bytes_utf8"`
}

type V2Data struct {
	ResponseStr string `json:"response_str"`
}

func (sc Scan) DataString() (string, error) {
	switch sc.DataVersion {
	case DataVersion_1:
		if !utf8.Valid(sc.Data.ResponseBytesUtf8) {
			return "", fmt.Errorf("%w: received invalid UTF-8 data", ErrData)
		}
		return string(sc.Data.ResponseBytesUtf8), nil
	case DataVersion_2:
		return sc.Data.V2Data.ResponseStr, nil
	default:
		return "", fmt.Errorf("%w: unknown data version number (%d)", ErrData, sc.DataVersion)
	}
}

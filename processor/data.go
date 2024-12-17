package processor

import (
	"errors"
	"fmt"
)

var ErrData = errors.New("data error:")

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

func (sc Scan) DataBytes() ([]byte, error) {
	switch sc.DataVersion {
	case DataVersion_1:
		return sc.Data.ResponseBytesUtf8, nil
	case DataVersion_2:
		return []byte(sc.Data.V2Data.ResponseStr), nil
	default:
		return nil, fmt.Errorf("%w: unknown data version number (%d)", ErrData, sc.DataVersion)
	}
}

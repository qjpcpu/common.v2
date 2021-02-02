package json

import (
	"bytes"
	sysjson "encoding/json"

	jsoniter "github.com/json-iterator/go"
	"github.com/qjpcpu/qjson"
)

type RawMessage = sysjson.RawMessage

var jiter = jsoniter.ConfigFastest

// PrettyMarshal colorful json
func PrettyMarshal(v interface{}) []byte {
	return qjson.PrettyMarshal(v)
}

// PrettyMarshalWithIndent colorful json
func PrettyMarshalWithIndent(v interface{}) []byte {
	return qjson.PrettyMarshalWithIndent(v)
}

// Marshal disable html escape
func Marshal(v interface{}) ([]byte, error) {
	return jiter.Marshal(v)
}

// Unmarshal same as sys unmarshal
func Unmarshal(data []byte, v interface{}) error {
	return jiter.Unmarshal(data, v)
}

// MustMarshal must marshal successful
func MustMarshal(v interface{}) []byte {
	data, err := Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// UnsafeMarshal marshal without error
func UnsafeMarshal(v interface{}) []byte {
	data, err := Marshal(v)
	if err != nil {
		return []byte("")
	}
	return data
}

// UnsafeMarshalString marshal without error
func UnsafeMarshalString(v interface{}) string {
	data, err := Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// UnsafeMarshalIndent marshal without error
func UnsafeMarshalIndent(v interface{}) []byte {
	data, err := Marshal(v)
	if err != nil {
		return []byte("")
	}
	var out bytes.Buffer
	sysjson.Indent(&out, data, "", "\t")
	return out.Bytes()
}

// MustUnmarshal must unmarshal successful
func MustUnmarshal(data []byte, v interface{}) {
	if err := Unmarshal(data, v); err != nil {
		panic(err)
	}
}

// DecodeJSONP 剔除jsonp包裹层
func DecodeJSONP(str []byte) []byte {
	var start, end int
	for i := 0; i < len(str); i++ {
		if str[i] == '(' {
			start = i
			break
		}
	}
	for i := len(str) - 1; i >= 0; i-- {
		if str[i] == ')' {
			end = i
			break
		}
	}
	if end > 0 {
		return str[start+1 : end]
	} else {
		return str
	}
}

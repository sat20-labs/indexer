package ord0_14_1

import "bytes"

type Tag struct {
	Value []byte
}

func (t Tag) Bytes() []byte {
	return t.Value
}

func (t Tag) IsChunked() bool {
	return bytes.Equal(t.Value, METADATA_TAG.Bytes())
}

func (t Tag) removeField(fields map[string][][]byte) []byte {
	tagStr := string(t.Bytes())
	values, ok := fields[tagStr]
	if !ok {
		return nil
	}
	if t.IsChunked() {
		var result []byte = nil
		for _, chunk := range values {
			result = append(result, chunk...)
		}
		delete(fields, tagStr)
		return result
	} else {
		if len(values) == 0 {
			return nil
		}
		value := values[0]
		values = values[1:]
		fields[tagStr] = values

		if len(values) == 0 {
			delete(fields, tagStr)
		}
		return value
	}
}

var (
	CONTENT_TYPE_TAG     = Tag{[]byte{0x01}}
	POINTER_TAG          = Tag{[]byte{0x02}}
	PARENT_TAG           = Tag{[]byte{0x03}}
	METADATA_TAG         = Tag{[]byte{0x05}}
	METAPROTOCOL_TAG     = Tag{[]byte{0x07}}
	CONTENT_ENCODING_TAG = Tag{[]byte{0x09}}
	DELEGATE_TAG         = Tag{[]byte{0x0B}}
)

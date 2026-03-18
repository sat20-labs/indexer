package ord0_14_1

import "bytes"

type Tag struct {
	Value []byte
}

func (t Tag) Bytes() []byte {
	return t.Value
}

func (t Tag) IsChunked() bool {
	return bytes.Equal(t.Value, METADATA_TAG.Bytes()) || bytes.Equal(t.Value, PROPERTIES_TAG.Bytes())
}

func (t Tag) take(fields map[string][][]byte) []byte {
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


func (t Tag) takeArray(fields map[string][][]byte) [][]byte {
	tagStr := string(t.Bytes())
	values, ok := fields[tagStr]
	if !ok {
		return nil
	}
	delete(fields, tagStr)	
	return values
}


var (
	CONTENT_TYPE_TAG     = Tag{[]byte{0x01}}
	POINTER_TAG          = Tag{[]byte{0x02}}
	PARENT_TAG           = Tag{[]byte{0x03}}
	METADATA_TAG         = Tag{[]byte{0x05}}
	METAPROTOCOL_TAG     = Tag{[]byte{0x07}}
	CONTENT_ENCODING_TAG = Tag{[]byte{0x09}}
	DELEGATE_TAG         = Tag{[]byte{0x0B}}
	RUNE_NAME_TAG        = Tag{[]byte{0x0D}}

	NOTE_TAG              = Tag{[]byte{15}} // 0.26
    PROPERTIES_TAG        = Tag{[]byte{17}} // 0.26
    PROPERTY_ENCODING_TAG = Tag{[]byte{19}} // 0.26
)

package ord

import (
	"encoding/json"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/ord/ord0_14_1"
)

type InscriptionResult = ord0_14_1.InscriptionResult

func GetProtocol(insc *InscriptionResult) (string, []byte) {
	content := insc.Inscription.Body

	var raw map[string]json.RawMessage
	err := json.Unmarshal([]byte(content), &raw)
	if err == nil {
		if v, ok := raw["p"]; ok {
			var p string
			if err := json.Unmarshal(v, &p); err == nil {
				return p, content
			}
		}
	}

	protocol := insc.Inscription.Metaprotocol
	if protocol != nil {
		jsonStr, err := common.Cbor2json(insc.Inscription.Metadata)
		if err == nil {
			content = jsonStr
		}
		return string(protocol), content
	}

	return "", nil
}

func IsOrdXProtocol(insc *InscriptionResult) (string, bool) {
	var content string

	content = string(insc.Inscription.Body)
	protocol := insc.Inscription.Metaprotocol
	if len(protocol) != 0 {
		if string(protocol) == common.PROTOCOL_NAME_ORDX {
			jsonStr, err := common.Cbor2json(insc.Inscription.Metadata)
			if err != nil {
				return content, false
			} else {
				content = string(jsonStr)
			}
		}
	}

	var ordxContent common.OrdxBaseContent
	err := json.Unmarshal([]byte(content), &ordxContent)
	if err != nil {
		return content, false
	}

	return content, ordxContent.P == common.PROTOCOL_NAME_ORDX
}
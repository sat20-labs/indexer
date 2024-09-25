package common

import (
	"encoding/json"
	"fmt"
	"strings"
)

// other domains


type DomainContent struct {
	P  string      `json:"p,omitempty"`
    Op string      `json:"op,omitempty"`
	Name  string   `json:"name"`
}


func ParseDomainContent(content string) *DomainContent {
	var ret DomainContent
	err := json.Unmarshal([]byte(content), &ret)
	if err != nil {
		Log.Warnf("invalid json: %s, %v", content, err)
		return nil
	}
	ret.Name = strings.TrimSpace(ret.Name)
	return &ret
}


func ParseCommonContent(content string) *OrdxUpdateContentV2 {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(content), &data)
	if err != nil {
		Log.Warnf("invalid json: %s, %v", content, err)
		return nil
	}

	var ret OrdxUpdateContentV2
	ret.KVs = make(map[string]string)
	for key, value := range data {
		strValue := fmt.Sprintf("%v", value)
		strValue = strings.TrimSpace(strValue)
		switch key {
		case "p": ret.P = strValue
		case "op": ret.Op = strValue
		case "name": ret.Name = strValue
		default: ret.KVs[key] = strValue
		}
	}

	return &ret
}


func ParsePrimaryNameContent(content string) *PrimaryNameBaseContent {
	var ret PrimaryNameBaseContent
	err := json.Unmarshal([]byte(content), &ret)
	if err != nil {
		Log.Warnf("invalid json: %s, %v", content, err)
		return nil
	}
	ret.Name = strings.TrimSpace(ret.Name)
	return &ret
}

func ParseRegContent(content string) *OrdxRegContent {
	var ret OrdxRegContent
	err := json.Unmarshal([]byte(content), &ret)
	if err != nil {
		Log.Warnf("invalid json: %s, %v", content, err)
		return nil
	}
	ret.Name = PreprocessName(ret.Name)
	// if strings.Contains(ret.Ticker, " ") {
	// 	Log.Warnf("invalid ticker name: %s", ret.Ticker)
	// 	return nil
	// }
	return &ret
}

func ParseUpdateContent(content string) *OrdxUpdateContentV2 {
	ret2 := ParseCommonContent(content)
	if ret2 == nil {
		return nil
	}

	value, ok := ret2.KVs["kvs"]
	if ok && strings.Contains(value, "=") {
		// rarepizza使用OrdxUpdateContentV1来修改封面，只能保留
		var ret OrdxUpdateContentV1
		err := json.Unmarshal([]byte(content), &ret)
		if err != nil {
			Log.Errorf("Unmarshal %s failed. %v", content, err)
			return nil
		}
		
		ret2 = &OrdxUpdateContentV2{P:ret.P, Op:ret.Op, Name:ret.Name}
		ret2.KVs = make(map[string]string)
		for _, kv := range ret.KVs {
			parts := strings.Split(kv, "=")
			if len(parts) != 2 {
				continue
			}
			ret2.KVs[parts[0]] = parts[1]
		}
	}
	
	return ret2
}

package common


type KeyValueInDB struct {
	Value         string
	InscriptionId string
}

type NameInfo struct {
	Base *InscribeBaseContent
	
	// realtime info
	OwnerAddress string
	Utxo         string

	Id           int64
	Name         string
	KVs          map[string]*KeyValueInDB
}

type NameServiceStatus struct {
	Version    string
	NameCount  uint64
}


func (p *NameServiceStatus) Clone () *NameServiceStatus {
	c := &NameServiceStatus{
		Version: p.Version,
		NameCount: p.NameCount,
	}
	return c
}

package runestone

import (
	"encoding/json"

	"lukechampine.com/uint128"
)

type Lot struct {
	Value *uint128.Uint128
}

func NewLot(value *uint128.Uint128) *Lot {
	return &Lot{Value: value}
}

func (l Lot) N() *uint128.Uint128 {
	return l.Value
}

func (l Lot) Add(rhs *Lot) Lot {
	result := l.Value.Add(*rhs.Value)
	return Lot{Value: &result}
}

func (l *Lot) AddAssign(rhs *Lot) {
	*l = l.Add(rhs)
}

func (l Lot) AddUint128(rhs *uint128.Uint128) Lot {
	return l.Add(&Lot{Value: rhs})
}

func (l *Lot) AddAssignUint128(rhs *uint128.Uint128) {
	l.AddAssign(&Lot{Value: rhs})
}

func (l Lot) Sub(rhs Lot) Lot {
	result := l.Value.Sub(*rhs.Value)
	return Lot{Value: &result}
}

func (l *Lot) SubAssign(rhs Lot) {
	*l = l.Sub(rhs)
}

func (l Lot) Div(rhs *uint128.Uint128) Lot {
	value := l.Value.Div(*rhs)
	return Lot{Value: &value}
}

func (l Lot) Rem(rhs *uint128.Uint128) Lot {
	value := l.Value.Mod(*rhs)
	return Lot{Value: &value}
}

func (l Lot) Eq(rhs *uint128.Uint128) bool {
	return l.Value.Equals(*rhs)
}

func (l Lot) Cmp(rhs *uint128.Uint128) int {
	return l.Value.Cmp(*rhs)
}

func (l Lot) MarshalJSON() ([]byte, error) {
	if l.Value == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(l.Value.String())
}

func (l *Lot) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		l.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	value, err := uint128.FromString(s)
	if err != nil {
		return err
	}
	l.Value = &value
	return nil
}

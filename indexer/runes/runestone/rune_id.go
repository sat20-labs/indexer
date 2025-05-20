package runestone

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"lukechampine.com/uint128"
)

type RuneId struct {
	Block uint64
	Tx    uint32
}

func NewRuneId(block uint64, tx uint32) (*RuneId, error) {
	if block == 0 && tx > 0 {
		return nil, errors.New("block=0 but tx>0")
	}
	return &RuneId{Block: block, Tx: tx}, nil
}

func (r RuneId) Delta(next RuneId) (uint64, uint32, error) {
	if next.Block < r.Block {
		return 0, 0, fmt.Errorf("next block is less than current block")
	}
	block := next.Block - r.Block
	var tx uint32
	if block == 0 {
		if next.Tx < r.Tx {
			return 0, 0, fmt.Errorf("next tx is less than current tx")
		}
		tx = next.Tx - r.Tx
	} else {
		tx = next.Tx
	}
	return block, tx, nil
}

func (r RuneId) Next(block uint128.Uint128, tx uint128.Uint128) (*RuneId, error) {
	//check block overflow
	if block.Hi > 0 {
		return nil, fmt.Errorf("block overflow")
	}
	if tx.Hi > 0 || tx.Lo > math.MaxUint32 {
		return nil, fmt.Errorf("tx overflow")
	}
	newBlock := r.Block + block.Lo
	//check for overflow
	if newBlock < r.Block {
		return nil, fmt.Errorf("block overflow")

	}
	var newTx uint32
	if block.IsZero() {
		newTx = r.Tx + uint32(tx.Lo)
		//check for overflow
		if newTx < r.Tx {
			return nil, fmt.Errorf("tx overflow")
		}
	} else {
		newTx = uint32(tx.Lo)
	}
	runeId, err := NewRuneId(newBlock, newTx)
	if err != nil {
		return nil, err
	}
	return runeId, nil
}

// TODO 以后重新编译数据时，可以改成 %d:%d
func (r RuneId) Hex() string {
	return fmt.Sprintf("%x_%x", r.Block, r.Tx)
}

func (r RuneId) String() string {
	return fmt.Sprintf("%d:%d", r.Block, r.Tx)
}

func RuneIdFromHex(s string) (*RuneId, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		parts = strings.Split(s, "_") // 暂时兼容下老版本，以后去掉
		if len(parts) != 2 {
			return nil, ErrSeparator
		}
	}
	block, err := strconv.ParseUint(parts[0], 16, 64)
	if err != nil {
		return nil, ErrBlock(parts[0])
	}
	tx, err := strconv.ParseUint(parts[1], 16, 32)
	if err != nil {
		return nil, ErrTransaction(parts[1])
	}
	return NewRuneId(block, uint32(tx))
}

func RuneIdFromString(s string) (*RuneId, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		parts = strings.Split(s, "_") // 暂时兼容下老版本，以后去掉
		if len(parts) != 2 {
			return nil, ErrSeparator
		}
	}
	block, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, ErrBlock(parts[0])
	}
	tx, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil, ErrTransaction(parts[1])
	}
	return NewRuneId(block, uint32(tx))
}

var (
	ErrSeparator   = errors.New("missing separator")
	ErrBlock       = func(err string) error { return fmt.Errorf("invalid Block height:%s", err) }
	ErrTransaction = func(err string) error { return fmt.Errorf("invalid Transaction index:%s", err) }
)

func (r RuneId) Cmp(other RuneId) int {
	return uint128.New(uint64(r.Tx), r.Block).Cmp(uint128.New(uint64(other.Tx), other.Block))
}

func (s *RuneId) ToPb() *pb.RuneId {
	pbValue := &pb.RuneId{
		Block: s.Block,
		Tx:    s.Tx,
	}
	return pbValue
}

func (s *RuneId) FromPb(pbValue *pb.RuneId) {
	s.Block = pbValue.Block
	s.Tx = pbValue.Tx
}

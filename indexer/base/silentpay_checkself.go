package base

import (
	"bytes"
	"container/list"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// Sources supply independently parsed Bitcoin data, never a reconstruction
// from the very SP index being audited. The small interface is also the E2E
// fixture boundary; production uses the existing internal bitcoind adapter.
type spAuditSource interface {
	BlockHash(height int) (string,error)
	Block(hash string) (*wire.MsgBlock,error)
	Transaction(txid string) (*wire.MsgTx,error)
}

type spRPCAuditSource struct{}

func (spRPCAuditSource) BlockHash(height int) (string,error) {return getBlockHash(uint64(height))}
func (spRPCAuditSource) Block(hash string) (*wire.MsgBlock,error) {
	raw,err:=getRawBlock(hash);if err!=nil {return nil,err}
	if len(raw)>8*1024*1024 {return nil,fmt.Errorf("oversized Bitcoin block")}
	data,err:=hex.DecodeString(raw);if err!=nil {return nil,err}
	r:=bytes.NewReader(data);block:=new(wire.MsgBlock)
	if err:=block.Deserialize(r);err!=nil {return nil,err}
	if r.Len()!=0 || block.BlockHash().String()!=hash {return nil,fmt.Errorf("Bitcoin block identity mismatch")}
	if err:=spValidateOriginalBlock(block);err!=nil {return nil,err}
	return block,nil
}
func (spRPCAuditSource) Transaction(txid string) (*wire.MsgTx,error) {
	if bitcoin_rpc.ShareBitconRpc==nil {return nil,fmt.Errorf("Bitcoin transaction source unavailable")}
	raw,err:=bitcoin_rpc.ShareBitconRpc.GetRawTx(txid);if err!=nil {return nil,err}
	if len(raw)>8*1024*1024 {return nil,fmt.Errorf("oversized Bitcoin transaction")}
	tx,err:=bitcoin_rpc.DecodeMsgTx(raw);if err!=nil {return nil,err}
	if tx.TxHash().String()!=txid {return nil,fmt.Errorf("Bitcoin transaction identity mismatch")}
	return tx,nil
}

func spValidateOriginalBlock(block *wire.MsgBlock) error {
	if block==nil || len(block.Transactions)==0 {return fmt.Errorf("empty Bitcoin block")}
	leaves:=make([]chainhash.Hash,len(block.Transactions))
	for i,tx:=range block.Transactions {if tx==nil {return fmt.Errorf("nil Bitcoin transaction")};leaves[i]=tx.TxHash()}
	for len(leaves)>1 {
		next:=make([]chainhash.Hash,0,(len(leaves)+1)/2)
		for i:=0;i<len(leaves);i+=2 {
			j:=i+1;if j==len(leaves) {j=i}
			var pair [64]byte;copy(pair[:32],leaves[i][:]);copy(pair[32:],leaves[j][:])
			next=append(next,chainhash.DoubleHashH(pair[:]))
		}
		leaves=next
	}
	if leaves[0]!=block.Header.MerkleRoot {return fmt.Errorf("Bitcoin transaction merkle root mismatch")}
	return blockchain.ValidateWitnessCommitment(btcutil.NewBlock(block))
}

type spCachedParent struct {txid string;outputs []*wire.TxOut;bytes int}
type spParentCache struct {limit,used int;order *list.List;items map[string]*list.Element}
func newSPParentCache() *spParentCache {return &spParentCache{limit:64*1024*1024,order:list.New(),items:make(map[string]*list.Element)}}
func (c *spParentCache) get(id string) []*wire.TxOut {
	if entry:=c.items[id];entry!=nil {c.order.MoveToFront(entry);return entry.Value.(spCachedParent).outputs};return nil
}
func (c *spParentCache) put(id string,outputs []*wire.TxOut) {
	size:=128+len(outputs)*32;for _,out:=range outputs {size+=len(out.PkScript)}
	if size>c.limit {return}
	if old:=c.items[id];old!=nil {c.order.MoveToFront(old);return}
	for c.used+size>c.limit && c.order.Len()>0 {
		last:=c.order.Back();entry:=last.Value.(spCachedParent);delete(c.items,entry.txid);c.used-=entry.bytes;c.order.Remove(last)
	}
	copied:=make([]*wire.TxOut,len(outputs));for i,out:=range outputs {copied[i]=&wire.TxOut{Value:out.Value,PkScript:append([]byte(nil),out.PkScript...)}}
	c.items[id]=c.order.PushFront(spCachedParent{txid:id,outputs:copied,bytes:size});c.used+=size
}

func spCandidateFromOriginal(tx *wire.MsgTx,index uint32,resolve func(wire.OutPoint)(*wire.TxOut,error)) (*SPTransaction,error) {
	outputs:=make([]SPOutput,0)
	for i,out:=range tx.TxOut {if common.SPP2TROutput(out.PkScript) {outputs=append(outputs,SPOutput{Vout:uint32(i),OutputKey:append([]byte(nil),out.PkScript[2:]...)})}}
	if index==0 || len(outputs)==0 {return nil,nil}
	inputs:=make([]common.SPInput,len(tx.TxIn))
	for i,in:=range tx.TxIn {
		previous,err:=resolve(in.PreviousOutPoint);if err!=nil {return nil,err}
		inputs[i]=common.SPInput{Outpoint:in.PreviousOutPoint,Prevout:previous,SignatureScript:in.SignatureScript,Witness:in.Witness}
	}
	tweak,eligible,err:=common.SPComputeTweak(inputs);if err!=nil {return nil,err};if !eligible {return nil,nil}
	return &SPTransaction{TxIndex:index,TxID:tx.TxHash().String(),Tweak:append([]byte(nil),tweak[:]...),P2TROutputs:outputs},nil
}

// audit enumerates original blocks, not Archive keys. It therefore detects
// whole missing transactions, a missing Archive+Live pair, and omitted fully
// spent historical transactions. It does not mutate the database under audit.
func (s *silentPayState) audit(ctx context.Context,expectedAnchor SPAnchor,source spAuditSource) error {
	view,anchor,err:=s.freshView(ctx);if err!=nil {return err};defer view.Close()
	if anchor!=expectedAnchor {return fmt.Errorf("%w: CheckSelf requires one committed Base anchor",ErrSPCorrupt)}
	canonical,err:=source.BlockHash(anchor.Height)
	if err!=nil || canonical!=anchor.BlockHash {return fmt.Errorf("CheckSelf canonical anchor unavailable or changed: %v",err)}
	cache:=newSPParentCache()
	var expectedLiveTx,expectedLiveOutputs uint64
	previousHash:=""
	if s.startHeight>0 && s.startHeight<=anchor.Height {
		previousHash,err=source.BlockHash(s.startHeight-1);if err!=nil {return err}
	}
	for height:=s.startHeight;height<=anchor.Height;height++ {
		if err:=ctx.Err();err!=nil {return err}
		hash,e:=source.BlockHash(height);if e!=nil {return fmt.Errorf("SP CheckSelf block %d: %w",height,e)}
		original,e:=source.Block(hash);if e!=nil {return fmt.Errorf("SP CheckSelf block %d: %w",height,e)}
		if original.BlockHash().String()!=hash || (height>s.startHeight || previousHash!="") && original.Header.PrevBlock.String()!=previousHash {return fmt.Errorf("SP CheckSelf block continuity %d",height)}
		if e=spValidateOriginalBlock(original);e!=nil {return fmt.Errorf("SP CheckSelf block %d: %w",height,e)}
		previousHash=hash
		current:=make(map[string]*wire.MsgTx,len(original.Transactions))
		for _,tx:=range original.Transactions {current[tx.TxHash().String()]=tx}
		resolve:=func(op wire.OutPoint)(*wire.TxOut,error){
			id:=op.Hash.String();var outputs []*wire.TxOut
			if tx:=current[id];tx!=nil {outputs=tx.TxOut} else {outputs=cache.get(id)}
			if outputs==nil {
				tx,e:=source.Transaction(id);if e!=nil {return nil,e}
				if tx.TxHash()!=op.Hash {return nil,fmt.Errorf("SP CheckSelf parent identity mismatch")}
				outputs=tx.TxOut;cache.put(id,outputs)
			}
			if uint64(op.Index)>=uint64(len(outputs)) {return nil,fmt.Errorf("SP CheckSelf prevout outside transaction")}
			return outputs[op.Index],nil
		}
		expected:=SPBlock{Height:height,BlockHash:hash,Transactions:make([]SPTransaction,0)}
		for index,tx:=range original.Transactions {
			if index==0 {continue}
			candidate,e:=spCandidateFromOriginal(tx,uint32(index),resolve)
			if e!=nil {return fmt.Errorf("SP CheckSelf %d/%d: %w",height,index,e)}
			if candidate==nil {continue}
			expected.Transactions=append(expected.Transactions,*candidate)
			stored,e:=spReadTransaction(view,height,uint32(index));if e!=nil {return e}
			if !spEqualTransactions(stored,*candidate) {return fmt.Errorf("%w: SP CheckSelf tx mismatch %d/%d",ErrSPCorrupt,height,index)}
			bits:=make([]byte,(len(candidate.P2TROutputs)+7)/8)
			for i,out:=range candidate.P2TROutputs {
				outpoint:=candidate.TxID+":"+spVoutString(out.Vout)
				raw,e:=view.Read(indexdb.GetUTXODBKey(outpoint))
				if errors.Is(e,common.ErrKeyNotFound) {continue};if e!=nil {return e}
				var utxo common.UtxoValueInDB
				if e=indexdb.DecodeBytesWithProto3(raw,&utxo);e!=nil {return e}
				if utxo.UtxoId!=common.ToUtxoId(height,index,int(out.Vout)) || utxo.Value!=tx.TxOut[out.Vout].Value {return fmt.Errorf("%w: Base identity/value %s",ErrSPCorrupt,outpoint)}
				address,e:=view.Read(indexdb.GetAddressIdKey(utxo.AddressId));if e!=nil {return e}
				script,e:=common.GetPkScriptFromAddress(string(address));if e!=nil {return e}
				if !common.SPP2TROutput(script) || !bytes.Equal(script[2:],out.OutputKey) {return fmt.Errorf("%w: Base script %s",ErrSPCorrupt,outpoint)}
				bits[i/8]|=1<<uint(i%8);expectedLiveOutputs++
			}
			live,e:=view.Read(spPositionKey(spLivePrefix,height,uint32(index)))
			if spBitmapEmpty(bits) {
				if !errors.Is(e,common.ErrKeyNotFound) {return fmt.Errorf("%w: fully spent tx remains Live %d/%d",ErrSPCorrupt,height,index)}
			} else {
				if e!=nil || !bytes.Equal(live,bits) {return fmt.Errorf("%w: Live/Base bitmap %d/%d",ErrSPCorrupt,height,index)}
				expectedLiveTx++
			}
		}
		want,e:=spEncodeBlock(expected);if e!=nil {return e}
		got,e:=view.Read(spBlockKey(height));if e!=nil || !bytes.Equal(got,want) {return fmt.Errorf("%w: SP CheckSelf block coverage %d",ErrSPCorrupt,height)}
		// Reject archive rows omitted from the block directory as well.
		prefix:=spPositionKey(spArchivePrefix,height,0)[:len(spArchivePrefix)+4]
		seen:=0
		e=view.Scan(common.ScanOptions{Prefix:prefix,KeysOnly:true},func(key,_ []byte)error{
			h,index,e:=spParsePosition(spArchivePrefix,key);if e!=nil || h!=height {return ErrSPCorrupt}
			i:=sort.Search(len(expected.Transactions),func(i int)bool{return expected.Transactions[i].TxIndex>=index})
			if i==len(expected.Transactions) || expected.Transactions[i].TxIndex!=index {return fmt.Errorf("%w: orphan archive row",ErrSPCorrupt)}
			seen++;return nil
		})
		if e!=nil || seen!=len(expected.Transactions) {return fmt.Errorf("SP CheckSelf archive rows %d: %v",height,e)}
		if height%10000==0 {common.Log.Infof("SP CheckSelf at %d, live transactions %d, live outputs %d",height,expectedLiveTx,expectedLiveOutputs)}
	}
	// Strict bounded passes reject extra future/pre-coverage keys. Pointwise
	// comparisons above already established exact membership, not just counts.
	var liveCount uint64
	err=view.Scan(common.ScanOptions{Prefix:spLivePrefix},func(key,bits []byte)error{
		if err:=ctx.Err();err!=nil {return err}
		h,index,e:=spParsePosition(spLivePrefix,key);if e!=nil || h<s.startHeight || h>anchor.Height || index==0 || spBitmapEmpty(bits) {return ErrSPCorrupt}
		liveCount++;return nil
	})
	if err!=nil {return err};if liveCount!=expectedLiveTx {return fmt.Errorf("%w: orphan Live rows",ErrSPCorrupt)}
	for _,prefix:=range [][]byte{spArchivePrefix,spBlockPrefix} {
		err=view.Scan(common.ScanOptions{Prefix:prefix,KeysOnly:true},func(key,_ []byte)error{
			if err:=ctx.Err();err!=nil {return err}
			if bytes.Equal(prefix,spBlockPrefix) {
				if len(key)!=len(prefix)+4 {return ErrSPCorrupt}
				key=append(append([]byte(nil),key...),0,0,0,0)
			}
			h,_,e:=spParsePosition(prefix,key)
			if e!=nil || h<s.startHeight || h>anchor.Height {return ErrSPCorrupt};return nil
		})
		if err!=nil {return err}
	}
	canonical,err=source.BlockHash(anchor.Height)
	if err!=nil || canonical!=anchor.BlockHash {return fmt.Errorf("SP CheckSelf anchor changed during verification: %v",err)}
	common.Log.Infof("SP CheckSelf passed through %d: live tx=%d outputs=%d",anchor.Height,expectedLiveTx,expectedLiveOutputs)
	return nil
}

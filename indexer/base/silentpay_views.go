package base

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
)

const spSnapshotTTL = time.Hour
const spMaxSnapshots = 4

type spPinnedView struct {
	meta SPSnapshot
	view common.DBReadView
	timer *time.Timer
	users int
	retired bool
}

// The pool owns public views, never wallet state. Timer expiry releases idle
// snapshots even if nobody makes another HTTP request. An in-flight page owns
// a lease, so expiry cannot close the database view underneath its decoder.
type spViewPool struct {
	mutex sync.Mutex
	views map[string]*spPinnedView
	cursorKey [32]byte
	closed bool
	initError error
}

func newSPViewPool() *spViewPool {
	p:=&spViewPool{views:make(map[string]*spPinnedView)}
	_,p.initError=rand.Read(p.cursorKey[:])
	return p
}

func (p *spViewPool) add(view common.DBReadView,anchor SPAnchor,start int) (SPSnapshot,error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.closed || p.initError!=nil {view.Close();return SPSnapshot{},ErrSPNotReady}
	for _,entry:=range p.views {
		if !entry.retired && entry.meta.Anchor==anchor && time.Now().Unix()<entry.meta.ExpiresAt {
			view.Close();return entry.meta,nil
		}
	}
	if len(p.views)>=spMaxSnapshots {view.Close();return SPSnapshot{},ErrSPSnapshotLimit}
	var nonce [16]byte
	if _,err:=rand.Read(nonce[:]);err!=nil {view.Close();return SPSnapshot{},err}
	id:=hex.EncodeToString(nonce[:])
	meta:=SPSnapshot{SnapshotID:id,Anchor:anchor,ExpiresAt:time.Now().Add(spSnapshotTTL).Unix(),CoverageStartHeight:start,SchemaVersion:SPSchemaVersion}
	entry:=&spPinnedView{meta:meta,view:view}
	p.views[id]=entry
	entry.timer=time.AfterFunc(spSnapshotTTL,func(){p.expire(id)})
	return meta,nil
}

func (p *spViewPool) expire(id string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	entry:=p.views[id]
	if entry==nil {return}
	entry.retired=true
	if entry.users==0 {entry.view.Close();delete(p.views,id)}
}

func (p *spViewPool) acquire(id string) (*spPinnedView,func(),error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	entry:=p.views[id]
	if p.closed || entry==nil || entry.retired || time.Now().Unix()>=entry.meta.ExpiresAt {return nil,nil,ErrSPSnapshotExpired}
	entry.users++
	var once sync.Once
	release:=func(){once.Do(func(){
		p.mutex.Lock();defer p.mutex.Unlock()
		entry.users--
		if entry.retired && entry.users==0 {entry.view.Close();delete(p.views,id)}
	})}
	return entry,release,nil
}

// close is invoked under IndexerMgr's RPC admission write barrier. All active
// API calls have completed at that point; Close also tolerates repeat calls.
func (p *spViewPool) close() {
	if p==nil {return}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.closed=true
	for id,entry:=range p.views {
		entry.retired=true
		entry.timer.Stop()
		if entry.users==0 {entry.view.Close();delete(p.views,id)}
	}
}

type spCursor struct {
	Mode string `json:"m"`
	Snapshot string `json:"s,omitempty"`
	From int `json:"f"`
	To int `json:"t"`
	Anchor string `json:"a"`
	Height int `json:"h"`
	Index uint32 `json:"i"`
}

func (p *spViewPool) encodeCursor(cursor spCursor) (string,error) {
	body,err:=json.Marshal(cursor);if err!=nil {return "",err}
	mac:=hmac.New(sha256.New,p.cursorKey[:]);mac.Write(body)
	return base64.RawURLEncoding.EncodeToString(append(body,mac.Sum(nil)...)),nil
}

func (p *spViewPool) decodeCursor(encoded string,want spCursor) (spCursor,error) {
	if len(encoded)>2048 {return want,ErrSPInvalidRequest}
	raw,err:=base64.RawURLEncoding.DecodeString(encoded)
	if err!=nil || len(raw)<sha256.Size {return want,ErrSPInvalidRequest}
	body,signature:=raw[:len(raw)-sha256.Size],raw[len(raw)-sha256.Size:]
	mac:=hmac.New(sha256.New,p.cursorKey[:]);mac.Write(body)
	if !hmac.Equal(mac.Sum(nil),signature) {return want,fmt.Errorf("%w: stale or invalid cursor",ErrSPInvalidRequest)}
	var got spCursor
	if err:=json.Unmarshal(body,&got);err!=nil {return want,ErrSPInvalidRequest}
	if got.Mode!=want.Mode || got.Snapshot!=want.Snapshot || got.From!=want.From || got.To!=want.To || got.Anchor!=want.Anchor || got.Height<want.From || got.Height>want.To+1 {
		return want,fmt.Errorf("%w: cursor scope",ErrSPInvalidRequest)
	}
	return got,nil
}

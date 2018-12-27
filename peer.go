package bcache

import (
	"log"

	"github.com/weaveworks/mesh"
)

type peer struct {
	cc       *cache
	name     mesh.PeerName
	send     mesh.Gossip
	actionCh chan func()
	quitCh   chan struct{}
}

func newPeer(name mesh.PeerName, maxKeys int) (*peer, error) {
	cc, err := newCache(maxKeys)
	if err != nil {
		return nil, err
	}

	p := &peer{
		cc:       cc,
		name:     name,
		send:     nil, // must be registered
		actionCh: make(chan func()),
		quitCh:   make(chan struct{}),
	}
	go p.loop()
	return p, nil
}

// register the result of a mesh.Router.NewGossip.
func (p *peer) register(send mesh.Gossip) {
	p.actionCh <- func() {
		p.send = send
	}
}

func (p *peer) Gossip() mesh.GossipData {
	return p.cc.copy()
}

// OnGossip merges received data into state and returns "everything new
// I've just learnt", or nil if nothing in the received data was new.
func (p *peer) OnGossip(msg []byte) (delta mesh.GossipData, err error) {
	e, err := newEntryFromBuf(msg)
	if err != nil {
		return nil, err
	}

	delta = p.cc.mergeDelta(e)
	if delta == nil {
		log.Printf("[%v]OnGossip %v => delta %v", p.name, e, delta)
	} else {
		log.Printf("[%v]OnGossip %v => delta %v", p.name, e, delta.(*cache).cc)
	}
	return delta, nil

}

// OnGossipBroadcast merges received data into state and returns a
// representation of the received data (typically a delta) for further
// propagation.
func (p *peer) OnGossipBroadcast(src mesh.PeerName, update []byte) (received mesh.GossipData, err error) {
	if src == p.name {
		log.Printf("FROM ME DUDE")
		return nil, nil
	}
	e, err := newEntryFromBuf(update)
	if err != nil {
		return nil, err
	}

	received = p.cc.mergeReceived(e)
	if received == nil {
		log.Printf("[%v]OnGossipBroadcast(nil) %v => delta %v", p.name, e, received)
	} else {
		log.Printf("[%v] from %v OnGossipBroadcast %v => delta %v", p.name, src, e, received.(*cache).cc)
	}
	return

}
func (p *peer) OnGossipUnicast(src mesh.PeerName, msg []byte) error {
	log.Printf("[error]OnGossipUnicast unexpected call")
	return nil
}

// Increment the counter by one.
func (p *peer) Set(key, val interface{}) {
	c := make(chan struct{})
	p.actionCh <- func() {
		defer close(c)
		e := p.cc.Set(key, val)
		if p.send != nil {
			p.send.GossipBroadcast(e)
		} else {
			log.Printf("no sender configured; not broadcasting update right now")
		}
	}
	<-c
	return
}

func (p *peer) Get(key interface{}) (interface{}, bool) {
	return p.cc.cc.Get(key)
}
func (p *peer) loop() {
	for {
		select {
		case f := <-p.actionCh:
			f()
		case <-p.quitCh:
			return
		}
	}
	log.Printf("peer[%v] loop finished", p.name)
}

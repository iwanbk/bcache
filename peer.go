package bcache

import (
	"github.com/weaveworks/mesh"
)

type peer struct {
	cc       *cache
	name     mesh.PeerName
	send     mesh.Gossip
	actionCh chan func()
	quitCh   chan struct{}
	logger   Logger
}

func newPeer(name mesh.PeerName, maxKeys int, logger Logger) (*peer, error) {
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
		logger:   logger,
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
func (p *peer) OnGossip(buf []byte) (delta mesh.GossipData, err error) {
	msg, err := newMessageFromBuf(buf)
	if err != nil {
		return nil, err
	}

	delta = p.cc.mergeDelta(msg)
	if delta == nil {
		p.logger.Printf("[%v]OnGossip %v => delta %v", p.name, msg, delta)
	} else {
		p.logger.Printf("[%v]OnGossip %v => delta %v", p.name, msg, delta.(*cache).cc)
	}
	return delta, nil

}

// OnGossipBroadcast merges received data into state and returns a
// representation of the received data (typically a delta) for further
// propagation.
func (p *peer) OnGossipBroadcast(src mesh.PeerName, update []byte) (received mesh.GossipData, err error) {
	if src == p.name { // message from ourself, is it possible?
		return nil, nil
	}
	msg, err := newMessageFromBuf(update)
	if err != nil {
		return nil, err
	}

	received = p.cc.mergeReceived(msg)
	if received == nil {
		p.logger.Printf("[%v]OnGossipBroadcast(nil) %v => delta %v", p.name, msg, received)
	} else {
		p.logger.Printf("[%v] from %v OnGossipBroadcast %v => delta %v", p.name, src, msg, received.(*cache).cc)
	}
	return

}
func (p *peer) OnGossipUnicast(src mesh.PeerName, msg []byte) error {
	p.logger.Printf("[error]OnGossipUnicast unexpected call")
	return nil
}

// Increment the counter by one.
func (p *peer) Set(key, val string, expiredTimestamp int64) {
	c := make(chan struct{})

	p.actionCh <- func() {
		defer close(c)
		e := p.cc.Set(key, val, expiredTimestamp)
		if p.send != nil {
			p.send.GossipBroadcast(e)
		} else {
			p.logger.Printf("no sender configured; not broadcasting update right now")
		}
	}

	<-c // wait for it to be finished
}

func (p *peer) Get(key string) (string, bool) {
	return p.cc.Get(key)
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
}

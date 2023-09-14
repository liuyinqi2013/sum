package ping

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"gl.zego.im/goserver/wormhole/quality"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	protocolICMP     = 1
	protocolIPv6ICMP = 58
)

var (
	ipv4Proto = map[string]string{"icmp": "ip4:icmp", "udp": "udp4"}
	ipv6Proto = map[string]string{"icmp": "ip6:ipv6-icmp", "udp": "udp6"}
)

var id int64 = 0

func getID() int64 {
	return atomic.AddInt64(&id, 1)
}

func isIPv4(ip net.IP) bool {
	return len(ip.To4()) == net.IPv4len
}

type MyPingPack struct {
	seqId    int
	sample   *quality.Sample
	closeCh  chan struct{}
	pinger   *MyPinger
	sendTime time.Time
	recvTime time.Time
	rtt      time.Duration
}

func (pack *MyPingPack) serve() {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-pack.closeCh:
	case <-timer.C:
		pack.recvTime = time.Now()
		pack.pinger.onLoss(pack)
	}
}

func (pack *MyPingPack) close() {
	close(pack.closeCh)
}

type MyPinger struct {
	id         int
	seqIdx     uint16
	tracker    uint64
	calculator *quality.Calculator
	waitPacks  map[int]*MyPingPack
	closeCh    chan struct{}
	addr       string
	conn       packetConn
	ipv4       bool
	protocol   string
	ipaddr     *net.IPAddr
	bodySize   int
	rwLock     sync.RWMutex
	rtt        time.Duration
	loss       float32
}

func NewMyPinger(addr string, protocol string) *MyPinger {
	id := getID()
	r := rand.New(rand.NewSource(id))
	return &MyPinger{
		id:         int(id),
		seqIdx:     65535,
		tracker:    r.Uint64(),
		calculator: quality.NewCalculator(),
		waitPacks:  make(map[int]*MyPingPack),
		closeCh:    make(chan struct{}),
		addr:       addr,
		conn:       nil,
		protocol:   protocol,
		bodySize:   16,
		ipaddr:     nil,
		rwLock:     sync.RWMutex{},
	}
}

func (p *MyPinger) onLoss(pack *MyPingPack) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	if _, ok := p.waitPacks[pack.seqId]; ok {
		p.calculator.Done(pack.recvTime, pack.sample, true)
		p.rtt, p.loss = p.calculator.GetQuality(pack.recvTime)
		delete(p.waitPacks, pack.seqId)
	}
}

func (p *MyPinger) Start() error {
	err := p.resolve()
	if err != nil {
		log.Printf("[E] resolve failed. err:%v", err.Error())
		return err
	}

	err = p.listen()
	if err != nil {
		log.Printf("[E] icmp listen packet failed. err:%v", err.Error())
		return err
	}

	if err := p.conn.SetFlagTTL(); err != nil {
		log.Printf("[E] set flag ttl failed. err:%v", err.Error())
		return err
	}

	return nil
}

func (p *MyPinger) Serve() {
	go p.recvICMP()
	defer p.conn.Close()

	for i := 0; i < quality.SampleCount/10; i++ {
		err := p.sendICMP()
		if err != nil {
			log.Printf("[E] send icmp failed. err:%v", err.Error())
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-p.closeCh:
			return
		case <-ticker.C:
			err := p.sendICMP()
			if err != nil {
				log.Printf("[E] send icmp failed. err:%v", err.Error())
				return
			}
		}
	}
}

func (p *MyPinger) GetQuality(now time.Time) (time.Duration, float32) {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.rtt, p.loss
}

func (p *MyPinger) sendICMP() error {
	var dst net.Addr = p.ipaddr
	if p.protocol == "udp" {
		dst = &net.UDPAddr{IP: p.ipaddr.IP, Zone: p.ipaddr.Zone}
	}

	data := uintToBytes(p.tracker)
	if padding := p.bodySize - 8; padding > 0 {
		data = append(data, bytes.Repeat([]byte{1}, padding)...)
	}

	body := &icmp.Echo{
		ID:   p.id,
		Seq:  int(p.seqIdx),
		Data: data,
	}

	msg := &icmp.Message{
		Type: p.conn.ICMPRequestType(),
		Code: 0,
		Body: body,
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	for {
		if _, err := p.conn.WriteTo(msgBytes, dst); err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
			return err
		}
		break
	}

	now := time.Now()
	pack := &MyPingPack{
		seqId:    int(p.seqIdx),
		sample:   p.calculator.AddSample(now),
		closeCh:  make(chan struct{}),
		pinger:   p,
		sendTime: now,
	}
	go pack.serve()

	p.rwLock.Lock()
	p.waitPacks[pack.seqId] = pack
	p.rwLock.Unlock()

	p.seqIdx++
	return nil
}

func (p *MyPinger) recvICMP() error {
	//var err error
	for {
		select {
		case <-p.closeCh:
			return nil
		default:
			/*
				if err = p.conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
					return err
				}
			*/

			len := 8 + p.bodySize
			data := make([]byte, len)
			n, _, _, err := p.conn.ReadFrom(data)
			log.Printf("[I] read from. addr:%v, err:%v, n:%v", p.addr, err, n)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						continue
					}
				}
				log.Printf("[E] read from failed. err:%v", err)
				return err
			}

			err = p.parseICMP(data)
			if err != nil {
				log.Printf("[E] parse icmp failed. err:%v", err)
			}
		}
	}
}

func (p *MyPinger) parseICMP(data []byte) error {
	var proto int
	var err error
	if p.ipv4 {
		proto = protocolICMP
	} else {
		proto = protocolIPv6ICMP
	}

	var m *icmp.Message
	if m, err = icmp.ParseMessage(proto, data); err != nil {
		return fmt.Errorf("error parsing icmp message: %v", err)
	}

	if m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply {
		log.Printf("[E] type error.")
		return nil
	}

	now := time.Now()
	switch pkt := m.Body.(type) {
	case *icmp.Echo:
		if p.protocol == "icmp" {
			if pkt.ID != p.id {
				log.Printf("[E] match id failed. addr:%v, send id:%v, recv id:%v", p.addr, p.id, pkt.ID)
				return nil
			}
		}

		if len(pkt.Data) < 8 {
			return fmt.Errorf("insufficient data received; got: %d %v", len(pkt.Data), pkt.Data)
		}

		tracker := bytesToUint(pkt.Data[:8])
		if tracker != p.tracker {
			log.Printf("[E] match tracker failed.")
			return nil
		}

		p.rwLock.RLock()
		pack, ok := p.waitPacks[pkt.Seq]
		p.rwLock.RUnlock()
		if !ok {
			return nil
		}

		pack.close()
		pack.recvTime = now
		pack.rtt = now.Sub(pack.sendTime)
		p.calculator.Done(now, pack.sample, false)
		rtt, loss := p.calculator.GetQuality(now)

		log.Printf("[I] recv icmp pack addr:%v, bytes:%d, seq:%v, rtt:%v, sid:%v, did:%v", p.addr, p.bodySize, pack.seqId, pack.rtt, p.id, pkt.ID)

		p.rwLock.Lock()
		defer p.rwLock.Unlock()
		delete(p.waitPacks, pack.seqId)
		p.rtt = rtt
		p.loss = loss

	default:
		return fmt.Errorf("invalid icmp echo reply; type: '%T', '%v'", pkt, pkt)
	}

	return nil
}

func (p *MyPinger) resolve() error {
	if len(p.addr) == 0 {
		return errors.New("addr cannot be empty")
	}

	ipaddr, err := net.ResolveIPAddr("ip", p.addr)
	if err != nil {
		return err
	}

	p.ipv4 = isIPv4(ipaddr.IP)
	p.ipaddr = ipaddr

	return nil
}

func (p *MyPinger) listen() error {
	var err error
	var conn packetConn

	if p.ipv4 {
		var c icmpv4Conn
		c.c, err = icmp.ListenPacket(ipv4Proto[p.protocol], "")
		conn = &c
	} else {
		var c icmpV6Conn
		c.c, err = icmp.ListenPacket(ipv6Proto[p.protocol], "")
		conn = &c
	}

	if err != nil {
		return err
	}
	p.conn = conn

	return nil
}

func bytesToUint(b []byte) uint64 {
	return uint64(binary.BigEndian.Uint64(b))
}

func uintToBytes(tracker uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, tracker)
	return b
}

func MyPingTest() {
	addrs := []string{"107.155.27.158"}

	f := func(addr string) {
		pinger := NewMyPinger(addr, "udp")
		err := pinger.Start()
		if err != nil {
			log.Printf("start failed. err:%v", err)
			return
		}

		go pinger.Serve()
		tick := time.NewTicker(2 * time.Second)
		for {
			<-tick.C
			rtt, loss := pinger.GetQuality(time.Now())
			log.Printf("addr:%v, rtt:%v, loss:%v", addr, rtt, loss)
		}
	}

	for _, addr := range addrs {
		go f(addr)
	}

	ch := make(chan bool)
	<-ch
}

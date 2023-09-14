package ping

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-ping/ping"
	"gl.zego.im/goserver/wormhole/quality"
)

type PingPack struct {
	seq      uint64
	sendTime time.Time
	sample   *quality.Sample
	pinger   *Pinger
	closeCh  chan struct{}
}

func (pack *PingPack) serve() {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		pack.pinger.OnLossPack(pack)
	case <-pack.closeCh:
	}
}

func (pack *PingPack) close() {
	close(pack.closeCh)
}

type Pinger struct {
	addr         string
	waitPacks    map[uint64]*PingPack
	closeCh      chan struct{}
	calculator   *quality.Calculator
	rwlock       sync.RWMutex
	lastRecvTime int64
}

func NewPinger(addr string) *Pinger {
	return &Pinger{
		addr:       addr,
		waitPacks:  make(map[uint64]*PingPack),
		closeCh:    make(chan struct{}),
		calculator: quality.NewCalculator(),
		rwlock:     sync.RWMutex{},
	}
}

func (p *Pinger) OnLossPack(pack *PingPack) {
	p.rwlock.Lock()
	defer p.rwlock.Unlock()

	if _, ok := p.waitPacks[pack.seq]; ok {
		log.Printf("[E] loss pack, seq:%v", pack.seq)
		delete(p.waitPacks, pack.seq)
		p.calculator.Done(time.Now(), pack.sample, true)
	}
}

func (p *Pinger) seqId(seqType int, seq int) uint64 {
	return uint64((seqType << 32) + seq)
}

func (p *Pinger) createPinger(interval time.Duration, count int, seqType int) (*ping.Pinger, error) {
	pinger, err := ping.NewPinger(p.addr)
	if err != nil {
		log.Printf("[E] create pinger failed. err:%v", err.Error())
		return nil, err
	}

	pinger.Count = count
	pinger.Interval = interval
	pinger.RecordRtts = false
	pinger.SetPrivileged(false)

	pinger.OnSend = func(pack *ping.Packet) {
		now := time.Now()
		pingPack := &PingPack{
			seq:      p.seqId(seqType, pack.Seq),
			sendTime: now,
			sample:   p.calculator.AddSample(now),
			pinger:   p,
			closeCh:  make(chan struct{}, 1),
		}
		go pingPack.serve()

		p.rwlock.Lock()
		defer p.rwlock.Unlock()
		p.waitPacks[pingPack.seq] = pingPack
	}

	pinger.OnRecv = func(pack *ping.Packet) {
		now := time.Now()
		seq := p.seqId(seqType, pack.Seq)
		log.Printf("[I] %d bytes from %s: icmp_seq=%d time=%v\n", pack.Nbytes, pack.IPAddr, seq, pack.Rtt)
		p.rwlock.RLock()
		pingPack, ok := p.waitPacks[seq]
		if !ok {
			p.rwlock.RUnlock()
			log.Printf("[E] not find pack. seq:%v, rtt:%v", pack.Seq, pack.Rtt)
			return
		}
		pingPack.close()
		p.rwlock.RUnlock()

		p.calculator.Done(now, pingPack.sample, false)
		atomic.StoreInt64(&p.lastRecvTime, now.Unix())

		p.rwlock.Lock()
		defer p.rwlock.Unlock()

		delete(p.waitPacks, pingPack.seq)
	}

	return pinger, nil
}

func (p *Pinger) Serve() {
	frequencyPinger, err := p.createPinger(20*time.Millisecond, quality.SampleCount/10, 1)
	if err != nil {
		log.Printf("[E] create requency pinger failed. err:%v", err.Error())
		return
	}

	pinger, err := p.createPinger(time.Second, 0, 0)
	if err != nil {
		log.Printf("[E] create pinger failed. err:%v", err.Error())
		return
	}

	errChan := make(chan error, 2)
	go func() {
		err := frequencyPinger.Run()
		if err != nil {
			errChan <- err
		}
	}()

	go func() {
		err := pinger.Run()
		if err != nil {
			errChan <- err
		}
	}()

	defer func() {
		frequencyPinger.Stop()
		pinger.Stop()
	}()

	select {
	case err = <-errChan:
		log.Printf("[E] pinger run failed. err:%v", err.Error())
	case <-p.closeCh:
		return
	}
}

func (p *Pinger) Close() {
	close(p.closeCh)
}

func (p *Pinger) GetQuality(now time.Time) (rtt time.Duration, loss float32) {
	return p.calculator.GetQuality(now)
}

func TestPing() {
	pinger := NewPinger("23.236.103.242")
	go pinger.Serve()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		<-tick.C
		rtt, loss := pinger.GetQuality(time.Now())
		log.Printf("ip:%v, rtt:%v, loss:%v", pinger.addr, rtt, loss)
	}
}

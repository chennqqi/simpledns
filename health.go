package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/chennqqi/go-ping"
	"github.com/miekg/dns"
)

type HealthChecker struct {
	interval  time.Duration
	iplist    []string
	healthMap sync.Map
	wg        sync.WaitGroup

	cancel context.CancelFunc
}

func ParseCheck(check string) (string, time.Duration) {
	check = strings.Trim(check, " ")
	check = strings.Trim(check, "\t")
	values := strings.Split(check, " ")
	if len(values) != 2 {
		return "", 0
	}
	to, _ := time.ParseDuration(values[1])
	return values[0], to
}

func NewHealthChecker(interval time.Duration) *HealthChecker {
	var h HealthChecker
	h.interval = interval
	return &h
}

func (this *HealthChecker) start(ctx context.Context) {
	list := this.iplist
	for i := 0; i < len(list); i++ {
		go this.check(ctx, list[i])
	}
}

func (this *HealthChecker) stop() {
	list := this.iplist
	this.cancel()
	oldlist := this.iplist
	for i := 0; i < len(list); i++ {
		this.healthMap.Delete(oldlist[i])
	}
	this.wg.Wait()
}

func (this *HealthChecker) check(ctx context.Context, ip string) {
	this.wg.Add(1)
	defer this.wg.Done()
	ticker := time.NewTicker(this.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p, err := ping.NewPinger(ip)
			if err != nil {
				logrus.Errorf("[health.go] new pinger error: %v", err)
				panic(err)
			}

			p.Count = 1
			p.Timeout = 3 * time.Second
			p.Run(ctx)
			statistics := p.Statistics()
			if statistics.PacketsSent == statistics.PacketsRecv {
				this.healthMap.Delete(ip)
			} else {
				this.healthMap.Store(ip, true)
			}
		}
	}
}

func (this *HealthChecker) Stop() {
	this.stop()
}

func (this *HealthChecker) Update(iplist []string) {
	this.Stop()
	this.iplist = iplist
	ctx, cancel := context.WithCancel(context.Background())
	this.cancel = cancel
	this.start(ctx)
}

func (this *HealthChecker) Filter(rrs []dns.RR) []dns.RR {
	var returnRrs = make([]dns.RR, 0)
	if rrs[0].Header().Rrtype != dns.TypeA && rrs[0].Header().Rrtype != dns.TypeA {
		return rrs
	}

	for i := 0; i < len(rrs); i++ {
		rr := rrs[i]
		switch rr.Header().Rrtype {
		//currently only ipv4, we don't certainly sure whether ping ipv6 is ok
		case dns.TypeA:
			if a, ok := rr.(*dns.A); ok {
				if _, exist := this.healthMap.Load(a.String()); !exist {
					returnRrs = append(returnRrs, rr)
				}
			}

		case dns.TypeAAAA:
			if a, ok := rr.(*dns.AAAA); ok {
				if _, exist := this.healthMap.Load(a.String()); !exist {
					returnRrs = append(returnRrs, rr)
				}
			}
		}
		//rr.Header().Rrtype == dns.TypeAAAA
	}
	return returnRrs
}

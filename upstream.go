package main

import (
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
)

//update by main.conf
type UpstreamServer struct {
	targets     []net.Conn
	index       int
	cached      bool
	cacheExpire time.Duration
	aCache      *cache.Cache
}

func NewUpstream(targets []string, cachedExpire time.Duration) (*UpstreamServer, error) {
	var up UpstreamServer

	for i := 0; i < len(targets); i++ {
		u, err := url.Parse(targets[i])
		if err != nil {
			logrus.Warnf("[server.go::UpstreamServer.NewUpstreamServer] parse upstream(%v), error: %v", targets[i], err)
			continue
		}
		switch u.Scheme {
		case "udp":
			if u.Port() != "" {
				conn, err := net.Dial("udp", u.Host)
				if err == nil {
					up.targets = append(up.targets, conn)
				} else {
					logrus.Errorf("[server.go::UpstreamServer.NewUpstreamServer] parse udp upstream(%v), error: %v", targets[i], err)
				}
			}
		default:
			logrus.Warnf("[server.go::UpstreamServer.NewUpstreamServer] parse upstream(%v), not support ignore", targets[i])
		}
	}
	if len(up.targets) == 0 {
		return nil, errors.New("no valid upstream server")
	}

	if cachedExpire != 0 {
		up.cached = true
		c := cache.New(cachedExpire, 2*cachedExpire)
		up.aCache = c
	}
	return &up, nil
}

func (s *UpstreamServer) Update([]byte) error {
	return nil
}

func (s *UpstreamServer) Close() {
	for i := 0; i < len(s.targets); i++ {
		s.targets[i].Close()
	}
	s.targets = make([]net.Conn, 0)
}

func (s *UpstreamServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	if s.cached && len(r.Question) >= 1 {
		//only response the first question
		for i := 0; i < 1; i++ {
			class := r.Question[i].Qclass
			name := r.Question[i].Name
			t := r.Question[i].Qtype
			if class == dns.ClassINET {
				switch t {
				case dns.TypeA:
					c := s.aCache
					v, exist := c.Get(name)
					if exist {
						if rr, ok := v.([]dns.RR); ok {
							m := new(dns.Msg)
							m.SetReply(r)
							m.Used(rr)
							w.WriteMsg(m)
							return
						}
					}
				}
			}
		}
	}

	//retry max
	index := s.index
	for retry := 0; retry < len(s.targets); retry++ {
		conn := s.targets[index]
		resp, err := dns.ExchangeConn(conn, r)
		if err != nil {
			logrus.Errorf("[server.go:ForwardServer.handleRequest] ExchangeConn error: %v", err)
			s.index = (index + 1) % len(s.targets)
		} else {
			if s.cached && len(r.Question) >= 1 {
				for i := 0; i < 1; i++ {
					q := &r.Question[i]
					//only cache class_INET and TypeA
					if q.Qclass == dns.ClassINET && q.Qtype == dns.TypeA {
						s.aCache.Set(q.Name, resp.Answer, cache.DefaultExpiration)
					}
				}
			}
			w.WriteMsg(resp)
			return
		}
	}
	//return empty
	m := new(dns.Msg)
	m.SetReply(r)
	w.WriteMsg(m)
}

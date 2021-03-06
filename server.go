package main

import (
	"context"
	"sync"
	"time"

	"github.com/chennqqi/goutils/consul.v2"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

type Server struct {
	tcpServer *dns.Server
	udpServer *dns.Server

	c       *consul.ConsulApp
	watcher *Watcher
	upStop  chan struct{}
}

func (s *Server) Init(cfg *Config) error {
	ms := make(map[string]DomainNameServer)
	var watchValues []NameValue
	udpServer := &dns.Server{
		// Address to listen on, ":dns" if empty.
		Addr: cfg.Addr, //""
		// if "tcp" or "tcp-tls" (DNS over TLS) it will invoke a TCP listener, otherwise an UDP one
		Net: "udp",

		// The net.Conn.SetReadTimeout value for new connections, defaults to 2 * time.Second.
		ReadTimeout: 5 * time.Second,
		// The net.Conn.SetWriteTimeout value for new connections, defaults to 2 * time.Second.
		WriteTimeout: 5 * time.Second,
		//	ReusePort bool
	}

	tcpServer := &dns.Server{
		// Address to listen on, ":dns" if empty.
		Addr: cfg.Addr, //""
		// if "tcp" or "tcp-tls" (DNS over TLS) it will invoke a TCP listener, otherwise an UDP one
		Net: "tcp",

		// The net.Conn.SetReadTimeout value for new connections, defaults to 2 * time.Second.
		ReadTimeout: 5 * time.Second,
		// The net.Conn.SetWriteTimeout value for new connections, defaults to 2 * time.Second.
		WriteTimeout: 5 * time.Second,
		//	ReusePort bool
	}

	//new server
	logrus.Println("servers:", cfg.Servers, len(cfg.Servers))
	for i := 0; i < len(cfg.Servers); i++ {
		sv := &cfg.Servers[i]
		ns, err := NewNameServer(sv)
		if err != nil {
			return err
		}
		_, exist := ms[sv.Name]
		if exist {
			logrus.Errorf("[server.go::Server.Init] ignore duplicate forward name: %v", sv.Name)
		} else {
			ms[sv.Name] = ns
			dns.Handle(sv.Name, ns)
			logrus.Println("Add handle", sv.Name, ns)
			for j := 0; j < len(sv.VZones); j++ {
				watchValues = append(watchValues, NameValue{
					sv.VZones[j].File,
					ns.servers[j],
				})
			}
		}
	}
	for i := 0; i < len(cfg.Forwards); i++ {
		sv := &cfg.Forwards[i]
		up, err := NewUpstream(sv.Upstreams, time.Duration(sv.CacheExpire))
		//not dynamic update
		if err != nil {
			logrus.Errorf("[server.go::Server.Init] NewUpstream error: %v", err)
		} else {
			_, exist := ms[sv.Name]
			if exist {
				logrus.Errorf("[server.go::Server.Init] ignore duplicate forward name: %v", sv.Name)
			} else {
				ms[sv.Name] = up
				dns.Handle(sv.Name, up)
			}
		}
	}
	watcher, err := NewWatcher(watchValues, gconsul.ConsulOperator)
	if err != nil {
		return err
	}

	s.watcher = watcher
	s.udpServer = udpServer
	s.tcpServer = tcpServer

	s.upStop = make(chan struct{})
	//new watcher
	return nil
}

func (s *Server) update() {
	watcher := s.watcher
	for {
		e, ok := <-watcher.Events()
		if !ok {
			logrus.Println("update stop")
			break
		}
		logrus.Infof("[server.go::Server.update] ProcessUpdate %v", e.Name)
		v := e.Extra
		logrus.Println("extra:", v)
		if v == nil {
			continue
		}
		if ds, ok := v.(DomainNameServer); ok {
			ds.Update(e.Data)
		} else {
			logrus.Panic("[server.go::update] assert type DomainNameServer error")
		}
	}
	close(s.upStop)
}

func (s *Server) Shutdown(ctx context.Context) error {
	defer func() {
		<-s.upStop
	}()
	s.watcher.Stop()

	var err1, err2 error
	if s.tcpServer != nil {
		err1 = s.tcpServer.ShutdownContext(ctx)
	}
	if s.udpServer != nil {
		err2 = s.udpServer.ShutdownContext(ctx)
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func (s *Server) Run() error {
	go s.update()
	go s.watcher.Run()

	var (
		wg         sync.WaitGroup
		err1, err2 error
	)

	{
		server := s.udpServer
		wg.Add(1)
		go func() {
			defer wg.Done()
			err1 = server.ListenAndServe()
		}()
	}
	{
		server := s.tcpServer
		wg.Add(1)
		go func() {
			defer wg.Done()
			err2 = server.ListenAndServe()
		}()
	}
	wg.Wait()

	if err1 != nil {
		return err1
	}
	return err2
}

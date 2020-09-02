package main

import (
	"context"
	"time"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

type App struct {
	server *dns.Server
	upStop chan struct{}
}

func (s *App) Init(cfg *Config) error {
	ms := make(map[string]DomainNameServer)
	server := &dns.Server{
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

	s.server = server
	s.upStop = make(chan struct{})
	//new watcher
	return nil
}

func (s *App) eventLoop() {
	for {
		//TODO:
	}
	close(s.upStop)
}

func (s *App) Shutdown(ctx context.Context) error {
	defer func() {
		<-s.upStop
	}()

	return s.server.ShutdownContext(ctx)
}

func (s *App) Run() error {
	go s.eventLoop()

	server := s.server
	return server.ListenAndServe()
}

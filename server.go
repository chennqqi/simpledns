package main

import (
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/chennqqi/goutils/consul"
	"github.com/miekg/dns"
)

type Server struct {
	serverMap map[string]DomainNameServer
	server    *dns.Server
	c         *consul.ConsulApp
	watcher   *Watcher
}

func (s *Server) Init(cfg *Config) error {
	ms := make(map[string]DomainNameServer)
	var watchValues []NameValue
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
		ns, err := NewNameServer(sv.VZones)
		logrus.Println("server:", ns, err)
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
			for j := 0; j < len(ns.servers); j++ {
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
	watcher, err := NewWatcher(watchValues)
	if err != nil {
		return err
	}

	s.watcher = watcher
	s.server = server
    s.serverMap  = ms
	//new watcher
	return nil
}

func (s *Server) update() {
	watcher := s.watcher
	for {
		e, ok := <-watcher.Events()
		if !ok {
			break
		}
		logrus.Infof("[server.go::Server.update] ProcessUpdate %v", e)

		v := e.Extra
		if v == nil {
			continue
		}
		if ds, ok := v.(DomainNameServer); ok {
			ds.Update(e.Data)
		}
	}
}

func (s *Server) Shutdown() error {
	//TODO: context
	s.watcher.Stop()
	s.server.Shutdown()
	return nil
}

func (s *Server) Run() error {
	for name, s := range s.serverMap {
		dns.Handle(name, s)
	}
	go s.watcher.Run()

	server := s.server
	return server.ListenAndServe()
}

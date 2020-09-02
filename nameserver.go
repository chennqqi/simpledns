package main

import (
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/yl2chen/cidranger"
)

//zone and nameserver interface
type DomainNameServer interface {
	ServeDNS(dns.ResponseWriter, *dns.Msg)
	Close()
	Update(txt []byte) error
}

/* a group of same name zone servers
   name server will choose the best match of zoneserver
*/
type NameServer struct {
	ServerConf

	vzones  []VZone
	servers []*ZoneServer
}

func NewNameServer(conf *ServerConf) (*NameServer, error) {
	var ns NameServer

	for i := 0; i < len(conf.VZones); i++ {
		z := &conf.VZones[i]
		zs := &ZoneServer{}
		zs.zone = z
		zs.roundRobin = conf.RoundRobin

		txt, err := ReadTxt(z.File)
		if err != nil {
			logrus.Errorf("[nameserver.go::NewNameServer] ReadTxt (%v) error: %v", z.File, err)
			return nil, err
		}
		err = zs.Update(txt)
		if err != nil {
			logrus.Errorf("[nameserver.go::NewNameServer] Update zone (%v) error: %v", z.File, err)
			return nil, err
		}

		ranger := cidranger.NewPCTrieRanger()
		for j := 0; j < len(z.MatchClients); j++ {
			ip := z.MatchClients[j]
			if strings.ToLower(ip) == "any" {
				ip = "0.0.0.0/0"
			}
			_, network, err := net.ParseCIDR(ip)
			if err != nil {
				logrus.Errorf("[nameserver.go::NewNameServer] ParseCIDR(%v) error: %v", ip, err)
				return nil, err
			}
			ranger.Insert(cidranger.NewBasicRangerEntry(*network))
		}
		zs.ranger = ranger
		ns.servers = append(ns.servers, zs)
	}
	ns.vzones = conf.VZones
	return &ns, nil
}

func (s *NameServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	//no locking code
	servers := s.servers
	//logrus.Println("servers count:", len(servers))
	for i := 0; i < len(servers); i++ {
		server := servers[i]
		addr := w.RemoteAddr()
		ip := addr.(*net.UDPAddr).IP
		//logrus.Println("remote IP:", ip)
		contain, _ := server.ranger.Contains(ip)
		//logrus.Println("remoteaddr:", addr, contain)
		//logrus.Println("remoteip:", ip, contain)
		if contain {
			server.ServeDNS(w, r)
			return
		}
	}
	//return empty
	//logrus.Infof("[nameserver.go::NameServer.ServeDNS] mis match ip")
	m := new(dns.Msg)
	m.SetReply(r)
	w.WriteMsg(m)
}

func (s *NameServer) Close() {
	//do nothing
}

func (s *NameServer) Update(txt []byte) error {
	//	for i := 0; i < len(s.vzones); i++ {
	//		//TODO:
	//	}
	return nil
}

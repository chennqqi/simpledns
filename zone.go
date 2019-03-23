package main

import (
	"bytes"

	"github.com/Sirupsen/logrus"
	"github.com/miekg/dns"
	"github.com/yl2chen/cidranger"
)

/*

class: IN 只internet, 常用IN和ANY
TYPE:


如果没有设定类，默认值为ANY。如果没有设定类型，默认值为ANY。
如果没有设定名称，默认值为”*”。
*/
type ZoneServer struct {
	zone   *VZone
	file   string
	name   string
	ranger cidranger.Ranger
	rrs    map[string][]dns.RR
}

/*
3.2.4 - Table Of Metavalues Used In Prerequisite Section

 CLASS    TYPE     RDATA    Meaning                    Function
 --------------------------------------------------------------
 ANY      ANY      empty    Name is in use             dns.NameUsed
 ANY      rrset    empty    RRset exists (value indep) dns.RRsetUsed
 NONE     ANY      empty    Name is not in use         dns.NameNotUsed
 NONE     rrset    empty    RRset does not exist       dns.RRsetNotUsed
 zone     rrset    rr       RRset exists (value dep)   dns.Used
The prerequisite section can also be left empty. If you have decided on the prerequisites you can tell what RRs should be added or deleted. The next table shows the options you have and what functions to call.
*/

func (s *ZoneServer) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	rrs := s.rrs
	m := new(dns.Msg)
	m.SetReply(r)
	if len(r.Question) >= 1 {
		//only response the first question
		for i := 0; i < 1; i++ {
			class := r.Question[i].Qclass
			name := r.Question[i].Name
			t := r.Question[i].Qtype
			rrArray := rrs[name]
			switch class {
			case dns.ClassANY:
				switch t {
				case dns.TypeANY:
					m.NameUsed(rrArray)
				default:
					m.RRsetUsed(rrArray)
				}
			case dns.ClassNONE:
				switch t {
				case dns.TypeANY:
					m.NameNotUsed(rrArray)
				default:
					m.RRsetNotUsed(rrArray)
				}
			default:
				switch t {
				case dns.TypeANY:
				default:
					m.Used(rrArray)
				}
			}
		}
	}
	w.WriteMsg(m)
}

func (s *ZoneServer) Update(txt []byte) error {
	rrs := make(map[string][]dns.RR)

	zp := dns.NewZoneParser(bytes.NewBuffer(txt), "", "")
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		if rr.Header().Class != dns.ClassINET {
			logrus.Warnf("[server.go:ZoneServer.Update] ignore not in class data: %v", rr)
			continue
		}

		rrArray, exist := rrs[rr.Header().Name]
		if exist {
			logrus.Println("rrarray", rrArray)
			logrus.Println("rr", rr)
			if rrArray[0].Header().Rrtype != rr.Header().Rrtype {
				logrus.Warnf("[server.go:ZoneServer.Update] ignore duplicate name(%v) type(%v), expect %v", rr.Header().Name,
					rr.Header().Rrtype, rrArray[0].Header().Rrtype)
				continue
			}
			rrArray = append(rrArray, rr)
			rrs[rr.Header().Name] = rrArray
		} else {
			var rrArray []dns.RR
			rrArray = append(rrArray, rr)
			rrs[rr.Header().Name] = rrArray
		}
	}

	if err := zp.Err(); err != nil {
		logrus.Warnf("[server.go::Zonserver.Update] zp.Err: %v", err)
		return err
	}

	//fix
	s.rrs = rrs
	return nil
}

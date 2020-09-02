package main

import (
	"bytes"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/yl2chen/cidranger"
)

/*
解析一个zone文件的A记录
class: IN 只internet, 常用IN和ANY
TYPE:

如果没有设定类，默认值为ANY。如果没有设定类型，默认值为ANY。
如果没有设定名称，默认值为”*”。

只支持A记录
*/
type ZoneServer struct {
	roundRobin bool
	checker    string
	zone       *VZone
	ranger     cidranger.Ranger
	rrs        map[string][]dns.RR
}

func (s *ZoneServer) rrPolicy(rr []dns.RR, r *dns.Msg) []dns.RR {
	//filter by ping checker
	//at least return one result, even if it wasn't check passed

	if s.roundRobin && len(rr) > 0 {
		shift := int(r.MsgHdr.Id) % len(rr)
		var nrr []dns.RR
		nrr = append(nrr, rr[shift:]...)
		nrr = append(nrr, rr[:shift]...)
		return nrr
	} else {
		return rr
	}
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

func (s *ZoneServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
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
					m.Used(s.rrPolicy(rrArray, r))
				}
			}
		}
	}
	w.WriteMsg(m)
}

func (s *ZoneServer) Close() {
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

	//update checker
	if s.checker != "" {
		var iplist []string
		for _, rr := range rrs {
			if rr[0].Header().Rrtype != dns.TypeA && rr[0].Header().Rrtype != dns.TypeAAAA {
				continue
			}
			if len(rr) == 1 {
				continue
			}
			switch rr[0].Header().Rrtype {
			case dns.TypeA:
				for i := 0; i < len(rr); i++ {
					//typeOfA := reflect.TypeOf(rr[0])
					//logrus.Println("RRTYPE:", typeOfA.Name(), typeOfA.Kind())
					if a, ok := rr[i].(*dns.A); ok {
						iplist = append(iplist, a.A.String())
					}
				}
				//currently only ipv4, we don't certainly sure whether ping ipv6 is ok
				//			case dns.TypeAAAA:
				//				if a, ok := rr.(*dns.AAAA); ok {
				//					iplist = append(a.A.String(), iplist)
				//				}
			}
		}
	}
	return nil
}

package main

import (
	"errors"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/chennqqi/goutils/utime"
)

type VZone struct {
	MatchClients []string `json:"match_clients" yaml:"match_clients"`
	Zone         string   `json:"zone" yaml:"zone"`
	File         string   `json:"file" yaml:"file"`
}

type ServerConf struct {
	Name       string  `json:"name" yaml:"name"`
	VZones     []VZone `json:"v_zones" yaml:"v_zones"`
	RoundRobin bool    `json:"round_robin" yaml:"round_robin"`
}

type UpstreamConf struct {
	Name        string         `json:"name" yaml:"name"`
	CacheExpire utime.Duration `json:"cache_expire" yaml:"cache_expire"`
	Upstreams   []string       `json:"upstreams" yaml:"upstreams"`
}

type WebConf struct {
	Addr     string `json:"addr" yaml:"addr"`
	Token    string `json:"token" yaml:"token"`
	Readonly bool   `json:"readonly" yaml:"readonly"`
}

type Config struct {
	Servers  []ServerConf   `json:"servers" yaml:"servers"`
	Forwards []UpstreamConf `json:"forwards" yaml:"forwards"`
	LogFile  string         `json:"log_file" yaml:"log_file"`
	LogLevel string         `json:"log_level" yaml:"log_level"`
	Addr     string         `json:"addr" yaml:"addr" default:""`
	Web      WebConf        `json:"web" yaml:"web"`
}

func ReadTxt(file string) ([]byte, error) {
	if strings.HasPrefix(file, "consul://") {
		u, err := url.Parse(file)
		if gconsul == nil {
			return nil, errors.New("consul not set")
		} else if err != nil {
			return nil, err
		} else if u.Path != "" {
			return gconsul.Get(u.Path[1:])
		} else {
			return nil, errors.New("consul path is nil")
		}
	} else {
		return ioutil.ReadFile(file)
	}
}

[![Build Status](https://travis-ci.org/chennqqi/simpledns.svg?branch=master)](https://travis-ci.org/chennqqi/simpledns) [![GoDoc](https://godoc.org/github.com/chennqqi/simpledns?status.svg)](https://godoc.org/github.com/chennqqi/simpledns)  [![LICENSE](https://img.shields.io/github/license/chennqqi/simpledns.svg?style=flat-square)](https://github.com/chennqqi/simpledns/blob/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/chennqqi/simpledns/go)](https://goreportcard.com/report/github.com/chennqqi/simpledns)

# simple dns server

	a simple dns server 
	
![](https://raw.githubusercontent.com/chennqqi/simpledns/master/screen.gif)

## feature

* consul based configure file
* standard zone file support
* dynamic load zone files
* client-ip(CIDR) based zone query
* outter dns server proxy with auto switch
* dns proxy with localcache, with this feature will enable proxy consul dns service to public
* multiple resolve with ping check

## build from source
	
required go version >= 1.10

	go get -u -v github.com/chennqqi/simpledns
 
## configure example

see [conf dir](https://github.com/chennqqi/simpledns/tree/master/conf)
	
# usage

	./simpledns -conf [configure uri]

## docker

build:
	./builddocker.sh
	
or pull:
	docker pull sort/simpledns

## known issue

* consul service not support udp check

## with consul as backend

add a forward upstream in forwards section, see [example](https://github.com/chennqqi/simpledns/tree/master/conf/simpledns.yml)

          - name: consul.
	        cache_expire: 5m
	        upstreams:
	        - 'udp://127.0.0.1:8600'

## with k8s/coredns as backend

         - name: cluster.local.
	        cache_expire: 5m
	        upstreams:
	        - 'udp://xx.xx.xx.xx:53'

## ping check

		servers: 
		- name: example.com.
		  v_zones:
		  - match_clients: [ "127.0.0.1/24" ]
		    file: 'conf/zones/t.example.com'
            checker: 'ping 10s' # every 10s check once

	note: if after ping check, this was no one valid ipaddr. it will ignore check result, return all result.
	

## TODO:

* control api, support ddns
* tcp dns service
* multiplex tcp check and tcp dns service
* add geo based zone query
* performance test

## License

Apache 2.0 license. See the [LICENSE](https://github.com/chennqqi/simpledns/blob/master/LICENSE) file for details.


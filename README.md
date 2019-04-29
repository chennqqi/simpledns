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


## TODO:

* tcp dns service
* multiplex tcp check and tcp dns service
* add geo based zone query
* performance test

## License

Apache 2.0 license. See the [LICENSE](https://github.com/chennqqi/simpledns/blob/master/LICENSE) file for details.


# simple dns server

## feature

* consul based configure file
* standard zone file support
* dynamic load zone files
* client-ip based zone query
* outter dns server proxy with auto switch

## build from source

	go get -u -v github.com/chennqqi/simpledns
 
## configure example

	see []()
	
# usage

	./simpledns -conf [configure uri]

## docker

	docker pull sort/simpledns

## known issue

* consul service not support udp check

## TODO:

* tcp dns service
* multiplex tcp check and tcp dns service
* add geo based zone query

## License

Apache 2.0

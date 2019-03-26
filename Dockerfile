FROM alpine

LABEL maintainer "https://github.com/chennqqi"

COPY . /go/src/github.com/chennqqi/simpledns
#RUN apk --update add --no-cache clamav ca-certificates
RUN apk --update add --no-cache ca-certificates
RUN apk --update add --no-cache -t .build-deps \
                    git \
                    go \
  && echo "Building godns docker deamon Go binary..." \
  && export GOPATH=/go \
  && mkdir -p /go/src/golang.org/x \
  && cd /go/src/golang.org/x \
  && git clone https://github.com/golang/net \
  && cd /go/src/github.com/chennqqi/godns \
  && go version \
  && go get \
  && go build -ldflags -o /bin/simpledns \
  && rm -rf /go /usr/local/go /usr/lib/go /tmp/* \
  && apk del --purge .build-deps

# Add hmb soft 
# Update ClamAV Definitions
#RUN hmb update

ENTRYPOINT ["simpledns"]
#ENTRYPOINT ["su-exec","malice","/sbin/tini","--","avscan"]
CMD ["--help"]


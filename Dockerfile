FROM malice/alpine

LABEL maintainer "https://github.com/chennqqi"

LABEL malice.plugin.repository = "https://github.com/chennqqi/hmbd.git"
LABEL malice.plugin.category="av"
LABEL malice.plugin.mime="*"
LABEL malice.plugin.docker.engine="*"

COPY . /go/src/github.com/chennqqi/hmbd
#RUN apk --update add --no-cache clamav ca-certificates
RUN apk --update add --no-cache ca-certificates
RUN apk --update add --no-cache -t .build-deps \
                    build-base \
                    mercurial \
                    musl-dev \
                    openssl \
                    bash \
                    wget \
                    git \
                    gcc \
                    go \
  && echo "Building hm webshell scanner deamon Go binary..." \
  && export GOPATH=/go \
  && mkdir -p /go/src/golang.org/x \
  && cd /go/src/golang.org/x \
  && git clone https://github.com/golang/net \
  && cd /go/src/github.com/chennqqi/hmbd \
  && go version \
  && go get \
  && go build -ldflags "-X main.Version=$(cat VERSION) -X main.BuildTime=$(date -u +%Y%m%d)" -o /bin/hmbd \
  && rm -rf /go /usr/local/go /usr/lib/go /tmp/* \
  && apk del --purge .build-deps


RUN chown malice -R /malware
WORKDIR /malware

# Add hmb soft 
ADD http://dl.shellpub.com/hmb/latest/hmb-linux-amd64.tgz /malware/hmb.tgz
RUN tar xvf /malware/hmb.tgz -C /malware
RUN ln -s /malware/hmb /bin/hmb

# Update ClamAV Definitions
#RUN hmb update

ENTRYPOINT ["hmbd"]
#ENTRYPOINT ["su-exec","malice","/sbin/tini","--","avscan"]
CMD ["--help"]

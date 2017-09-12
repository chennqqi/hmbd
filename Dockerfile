FROM malice/alpine

LABEL maintainer "https://github.com/chennqqi"

LABEL malice.plugin.repository = "https://github.com/chennqqi/hmd.git"
LABEL malice.plugin.category="av"
LABEL malice.plugin.mime="*"
LABEL malice.plugin.docker.engine="*"

COPY . /go/src/github.com/chennqqi/hmd
RUN apk --update add --no-cache clamav ca-certificates
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
  && cd /go/src/github.com/chennqqi/hmd \
  && export GOPATH=/go \
  && go version \
  && go get \
  && go build -ldflags "-X main.Version=$(cat VERSION) -X main.BuildTime=$(date -u +%Y%m%d)" -o /bin/hmd \
  && rm -rf /go /usr/local/go /usr/lib/go /tmp/* \
  && apk del --purge .build-deps

# Update ClamAV Definitions
RUN hmd update

# Add EICAR Test Virus File to malware folder
#ADD http://www.eicar.org/download/eicar.com.txt /malware/EICAR

RUN chown malice -R /malware

WORKDIR /malware

ENTRYPOINT ["su-exec","malice","/sbin/tini","--","avscan"]
CMD ["--help"]

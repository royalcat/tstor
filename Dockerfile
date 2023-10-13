#===============
# Stage 1: Build
#===============

FROM golang:1.20 as builder

ENV BIN_REPO=git.kmsign.ru/royalcat/tstor
ENV BIN_PATH=$GOPATH/src/$BIN_REPO

COPY . $BIN_PATH
WORKDIR $BIN_PATH

RUN apk add fuse-dev git gcc libc-dev g++ make

RUN BIN_OUTPUT=/bin/tstor make build

#===============
# Stage 2: Run
#===============

FROM alpine:3

RUN apk add gcc libc-dev fuse-dev

COPY --from=builder /bin/tstor /bin/tstor
RUN chmod +x /bin/tstor

RUN mkdir /tstor-data

RUN echo "user_allow_other" >> /etc/fuse.conf
ENV tstor_FUSE_ALLOW_OTHER=true

ENTRYPOINT ["./bin/tstor"]

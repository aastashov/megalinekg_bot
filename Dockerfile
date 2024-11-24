FROM golang:1.23.3-alpine3.20 AS builder

RUN apk add --no-cache git gcc g++

COPY . /srv

ARG PAT

RUN set -x \
    && echo "machine github.com login aastashov password ${PAT}" > ~/.netrc \
    && cd /srv/ \
    && go build -o app .


FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /srv/app /usr/local/bin/

CMD ["app"]

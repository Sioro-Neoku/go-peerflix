FROM golang

COPY . /usr/src/go-peerflix/
WORKDIR /usr/src/go-peerflix/

RUN go build .

ENTRYPOINT [ "./go-peerflix" ]

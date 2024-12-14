
##### builder -- target which actually builds the two binaries
FROM golang:1.22 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 ./build.sh


##### scanner -- the runnable "scanner" binary, as provided in the original repo
FROM alpine AS scanner
WORKDIR /app
COPY --from=builder /src/bin/scanner .
CMD ["/app/scanner"]

##### processor -- the runnable "takehome-processor" binary
FROM alpine AS processor
WORKDIR /app
COPY --from=builder /src/bin/takehome-processor .
CMD ["/app/takehome-processor", "-D", "--pretty", "server"]

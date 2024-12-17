
##### builder -- target which actually builds the two binaries
FROM golang:1.22 AS builder

WORKDIR /src

# Tool necessary for builds down the line
RUN go install github.com/google/wire/cmd/wire@latest

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY config config
COPY database database
COPY logging logging
COPY scanning scanning
COPY server server
COPY build.sh ./
RUN ./build.sh


##### scanner -- the runnable "scanner" binary, as provided in the original repo
FROM alpine AS scanner
WORKDIR /app
COPY --from=builder /src/bin/scanner .
CMD ["/app/scanner"]

##### processor -- the runnable "takehome-processor" binary
FROM alpine AS processor
WORKDIR /app
COPY --from=builder /src/bin/takehome-processor .
ENTRYPOINT [ "/app/takehome-processor" ]

FROM golang:1.21.1-bookworm AS builder

WORKDIR /src

COPY . .

RUN go mod download &&\
go build -trimpath -ldflags "-w -s" -o /build/app

# -----------------------------------------------

FROM debian:bookworm-slim as deploy

COPY --from=builder /build/app .

ENTRYPOINT ["./app"]

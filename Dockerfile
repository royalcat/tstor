FROM golang:1.21 as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./src ./src
COPY ./cmd ./cmd
COPY ./assets ./assets
COPY ./templates ./templates
COPY embed.go embed.go

RUN go generate ./...
RUN CGO_ENABLED=0 go build -tags timetzdata -o /tstor ./cmd/tstor/main.go 


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /tstor /tstor

ENTRYPOINT ["/tstor"]

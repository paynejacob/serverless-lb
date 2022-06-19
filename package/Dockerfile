FROM golang:1.18-alpine as operator-builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY main.go main.go
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=linux go build -a -o controller main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=operator-builder /workspace/controller /bin/controller
USER 65532:65532

ENTRYPOINT ["controller"]

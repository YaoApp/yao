FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /mock-llm .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /mock-llm /usr/local/bin/mock-llm
COPY fixtures/ /fixtures/
EXPOSE 9999
ENTRYPOINT ["mock-llm"]
CMD ["-port", "9999", "-fixtures", "/fixtures"]

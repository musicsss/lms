FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -o /lms ./cmd/lms/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata ffmpeg
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /lms .
COPY config.yaml .
EXPOSE 8080
CMD ["./lms"]

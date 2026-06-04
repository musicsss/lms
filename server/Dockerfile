FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata ffmpeg
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY bin/lms .
COPY config.yaml .

EXPOSE 8080
CMD ["./lms"]

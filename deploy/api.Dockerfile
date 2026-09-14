FROM golang:1.27.1 AS build
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/settle ./cmd/settle

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget
COPY --from=build /out/app /usr/local/bin/app
COPY --from=build /out/settle /usr/local/bin/settle
COPY configs /app/configs
WORKDIR /app
ENV TZ=Asia/Shanghai
EXPOSE 8000
ENTRYPOINT ["/usr/local/bin/app", "-conf", "/app/configs/config.yaml"]

# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci --legacy-peer-deps 2>/dev/null || npm install --legacy-peer-deps
COPY web/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.21-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
ARG GOPROXY=https://goproxy.cn,direct
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN sed -i "s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g" /etc/apk/repositories 2>/dev/null; true
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o zhangyi .

# Stage 3: Production
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=backend /app/zhangyi .
COPY templates/ ./templates/
RUN mkdir -p data backups exports
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/health || exit 1
CMD ["./zhangyi"]

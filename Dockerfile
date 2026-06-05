# =============================================
# Stage 1: Build frontend (Vue3 + TailwindCSS)
# =============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

RUN npm config set registry https://registry.npmmirror.com

COPY frontend/package*.json ./
RUN npm install

COPY frontend/ ./
RUN npm run build

# =============================================
# Stage 2: Build Go backend
# =============================================
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod ./
COPY main.go ./
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

RUN CGO_ENABLED=0 GOOS=linux go build -o webapp .

# =============================================
# Stage 3: Runtime
# =============================================
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=backend-builder /app/webapp .

EXPOSE 8808

CMD ["./webapp"]
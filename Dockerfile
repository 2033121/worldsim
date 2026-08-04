# WorldSim Dockerfile — 多阶段构建（Go 单二进制，零外部依赖）
# 构建：docker build -t worldsim .
# 运行：docker run -p 48091:48091 -p 48090:48090 -v $(pwd)/wsdata:/app/wsdata worldsim

# ---------- 阶段 1：构建 ----------
# 用 daocloud 镜像源前缀绕过 Docker Hub 直连失败（国内网络）
FROM docker.m.daocloud.io/library/golang:1.22-alpine AS builder

WORKDIR /src
COPY . .
# 静态编译，去掉调试信息缩小体积
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o worldsim .

# ---------- 阶段 2：运行 ----------
FROM docker.m.daocloud.io/library/alpine:3.20

# ca-certificates：HTTPS 调 LLM 必需；tzdata：时区；bash：entrypoint
RUN apk add --no-cache ca-certificates tzdata bash

WORKDIR /app
# 拷贝二进制
COPY --from=builder /src/worldsim /app/worldsim
# 拷贝运行时需要的素材（首次启动会复制到 wsdata）
COPY --from=builder /src/worldbooks /app/seed/worldbooks
COPY --from=builder /src/material /app/seed/material
# entrypoint
COPY scripts/docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh /app/worldsim

# 数据目录（挂载卷）
RUN mkdir -p /app/wsdata
VOLUME /app/wsdata

# 双端口：48091 世界模拟 WebUI / 48090 小说服务
EXPOSE 48091 48090

# 时区
ENV TZ=Asia/Shanghai

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/worldsim", "/app/wsdata"]

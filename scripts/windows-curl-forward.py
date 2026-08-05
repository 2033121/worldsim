#!/usr/bin/env python3
"""
Windows 宿主 curl.exe(Schannel) 转发服务
========================================
背景：中转站 tokenrhythm.studio 的 WAF/ALB 只放行 Windows 原生 Schannel TLS 指纹
（即 curl.exe），拒绝 Go / Python-OpenSSL / busybox 的 TLS ClientHello 指纹。
因此容器内应用无法直连中转站（无论 bridge 还是 host 网络）。

本脚本在 Windows 宿主起一个本地 HTTP 服务，收到容器应用的请求后，调用
curl.exe（Schannel TLS）转发到中转站，从而用宿主国内网络直连中转站。

用法：
    py scripts/windows-curl-forward.py [端口]     # 默认 49322
容器配置（api.json / docker-compose 用 host 网络即可）：
    base_url = http://127.0.0.1:<端口>/v1     # host 网络下 127.0.0.1 即宿主
退出：Ctrl+C 停止。
"""
import http.server
import os
import subprocess
import sys
import tempfile
import threading
import time

LISTEN_HOST = "0.0.0.0"
LISTEN_PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 49322
UPSTREAM = os.environ.get("UPSTREAM", "https://tokenrhythm.studio")
CURL = os.environ.get("CURL", "curl.exe")

# 透传白名单请求头（其余忽略，避免 Host/内网头干扰）
PASS_HEADERS = {
    "authorization", "content-type", "accept", "user-agent",
    "x-api-key", "x-request-id", "x-stainless-*", "openai-organization",
    "openai-project", "x-session-id", "accept-language",
}

# 响应头里不回传的（由本服务自行管理连接）
DROP_RESP_HEADERS = {"transfer-encoding", "connection", "keep-alive", "content-length"}


class ForwardHandler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def _forward(self):
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else b""
        print("REQ {} {} len={}".format(self.command, self.path, length), flush=True)

        # 请求体写入临时文件（curl --data-binary @file 避免命令行引号/长度问题）
        bf = tempfile.NamedTemporaryFile(delete=False)
        try:
            bf.write(body)
            bf.flush()
        finally:
            bf.close()

        # 响应头临时文件
        hf = tempfile.NamedTemporaryFile(delete=False)
        hf.close()

        cmd = [CURL, "-sS", "-N", "-k", "-X", self.command,
               "--data-binary", "@" + bf.name,
               "-D", hf.name,
               UPSTREAM + self.path]
        for k, v in self.headers.items():
            if k.lower() in PASS_HEADERS:
                cmd += ["-H", "{}: {}".format(k, v)]
        # 保证 Content-Type 一定有（中转站要求），避免 curl --data-binary 默认表单类型
        has_ct = any(h.lower() == "content-type" for h, _ in self.headers.items())
        if not has_ct:
            cmd += ["-H", "Content-Type: application/json"]

        env = dict(os.environ)
        env["NO_PROXY"] = "*"
        env["no_proxy"] = "*"
        try:
            proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                    env=env, bufsize=0)
        except Exception as e:
            self._send_simple(502, "failed to launch curl.exe: {}".format(e))
            os.unlink(bf.name)
            os.unlink(hf.name)
            return

        # 收集 curl stderr（诊断用）
        errbuf = []
        def _drain():
            while True:
                line = proc.stderr.readline()
                if not line:
                    break
                errbuf.append(line.decode("utf-8", "replace").rstrip())
        terr = threading.Thread(target=_drain, daemon=True)
        terr.start()

        # 等待响应头文件就绪（curl -D 会先写 header 再写 body）；
        # 大上下文/流式请求首 token 可能较慢，给足 300s（新世界提示词含角色名册+背景池，上下更大）。
        header = b""
        deadline = time.time() + 300
        while time.time() < deadline:
            try:
                with open(hf.name, "rb") as f:
                    header = f.read()
            except Exception:
                pass
            if b"\r\n\r\n" in header or b"\n\n" in header:
                break
            if proc.poll() is not None:
                break
            time.sleep(0.05)

        sep = b"\r\n\r\n" if b"\r\n\r\n" in header else b"\n\n"
        if sep not in header:
            # 没有拿到有效响应头
            proc.kill()
            proc.wait(timeout=5)
            print("NO-HDR path={} exit={} stderr={}".format(
                self.path, proc.returncode, " | ".join(errbuf[-5:])), flush=True)
            self._send_simple(502, "upstream returned no valid headers")
            os.unlink(bf.name)
            os.unlink(hf.name)
            return

        head, _ = header.split(sep, 1)
        lines = head.split(b"\r\n") if b"\r\n" in head else head.split(b"\n")

        status = 502
        resp_headers = []
        for ln in lines:
            if ln.startswith(b"HTTP/"):
                parts = ln.split(b" ")
                if len(parts) >= 2:
                    try:
                        status = int(parts[1])
                    except ValueError:
                        pass
            elif b":" in ln:
                k, v = ln.split(b":", 1)
                kn = k.strip().decode("latin-1", "replace").lower()
                vn = v.strip().decode("latin-1", "replace")
                if kn in DROP_RESP_HEADERS or kn.startswith("x-"):
                    continue
                resp_headers.append((kn, vn))

        self.send_response(status)
        for k, v in resp_headers:
            self.send_header(k, v)
        # 流式：不设 Content-Length，用 Connection: close 界定结束
        self.send_header("Connection", "close")
        self.end_headers()

        try:
            while True:
                chunk = proc.stdout.read(8192)
                if not chunk:
                    break
                self.wfile.write(chunk)
                self.wfile.flush()
        except (BrokenPipeError, ConnectionResetError):
            try:
                proc.kill()
            except Exception:
                pass
        finally:
            try:
                proc.stdout.close()
            except Exception:
                pass
            try:
                proc.wait(timeout=5)
            except Exception:
                try:
                    proc.kill()
                except Exception:
                    pass
            self.close_connection = True
            os.unlink(bf.name)
            os.unlink(hf.name)

    def _send_simple(self, status, msg):
        body = msg.encode("utf-8", "replace")
        self.send_response(status)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Connection", "close")
        self.end_headers()
        try:
            self.wfile.write(body)
            self.wfile.flush()
        except Exception:
            pass
        self.close_connection = True

    def _do(self):
        try:
            self._forward()
        except Exception as e:
            try:
                self._send_simple(502, "forward error: {}".format(e))
            except Exception:
                pass

    do_POST = _do
    do_GET = _do
    do_OPTIONS = _do
    do_PUT = _do
    do_DELETE = _do

    def log_message(self, *args):
        pass


def main():
    srv = http.server.ThreadingHTTPServer((LISTEN_HOST, LISTEN_PORT), ForwardHandler)
    srv.daemon_threads = True
    print("curl.exe forward listening on {0}:{1} -> {2}".format(
        LISTEN_HOST, LISTEN_PORT, UPSTREAM), flush=True)
    try:
        srv.serve_forever()
    except KeyboardInterrupt:
        pass
    print("stopped.", flush=True)


if __name__ == "__main__":
    main()
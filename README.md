# QCP Benchmark

**QCP vs KCP — 真实网络压测（拒绝 loopback Optimistic 路径）**

> KCP 在 `baseline_kcp.go`，**仅作压测基线**，不是 QCP 组件。

## 验证层级

| 层级 | 命令 | 网络 |
|------|------|------|
| **CI 快速门** | `go test -run Simulated` / `-verify` | 纯模型，无 socket、**无 loopback** |
| **最终 Docker** | `.\scripts\verify-docker.ps1` | `10.10.0.0/24` bridge + **tc/netem** |
| **最终本机网卡** | `.\scripts\verify-host.ps1` | 绑定 LAN IP，**拒绝 127.0.0.1** |

`-verify-net` 使用真实网络（`qcp-lib-go` :9003），**禁止 loopback**。  
判定：**P50 为主**（QCP ≤ KCP+5%）；丢包 ≥3% 场景额外检查 P99。  
每次验证会生成 `性能对比表格.md`，只保留 `QCP / TCP / KCP`，不包含 UDP。

## 快速 CI（仿真）

```bash
go test ./... -run Simulated -count=1
go run . -verify -duration 2s -connections 10
```

## 最终验证 — Docker（推荐）

```bash
bash scripts/verify-docker.sh 20 5s
```

- Server: `10.10.0.10`（`qcp-lib-go` 官方栈 `:9003`）
- Client: `10.10.0.20` + `netem.sh` 按场景注入 delay/loss
- `-verify-net` 拒绝 loopback
- 验证完成后会刷新 `性能对比表格.md`

单场景：

```bash
docker compose up -d server
docker compose run --rm -e SCENARIO=wifi client \
  -verify-net -server 10.10.0.10:9000 -scenario wifi -duration 5s
```

## 最终验证 — 本机网卡

```powershell
cd qcp-benchmark
.\scripts\verify-host.ps1 -Connections 20 -Duration 5s
```

```bash
bash scripts/verify-host.sh 20 5s
```

## 架构

| 端口 | 协议 |
|------|------|
| 9000 | TCP echo |
| 9001 | UDP echo |
| 9002 | KCP 基线 echo |
| 9003 | **QCP 官方库** (`qcp-lib-go`) |

## License

MIT

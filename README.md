# QCP Benchmark

QCP vs KCP vs TCP vs UDP — 2026 极致性能对比

## 快速开始

```bash
# Docker 测试（推荐）
docker-compose up -d server
docker-compose run --rm client -mode client -server 10.10.0.10:9000 -protocol all -duration 10s -connections 100 -loss 0.05

# 本地测试
go run . -mode all -duration 10s -connections 100 -loss 0.05
```

## 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-mode` | all | server, client, all |
| `-server` | - | 服务器地址 |
| `-protocol` | all | qcp, tcp, udp, kcp, all |
| `-duration` | 10s | 测试时长 |
| `-connections` | 100 | 并发连接数 |
| `-loss` | 0 | 丢包率 (0-1) |

## 延迟指标说明

| 指标 | 含义 |
|------|------|
| **P50** | 50% 请求低于此延迟（中位数） |
| **P95** | 95% 请求低于此延迟 |
| **P99** | 99% 请求低于此延迟（最差情况） |

**P99 最重要** — 玩家感知的是最慢的 1%。

## 实测结果

### QCP vs KCP 核心对比

| 丢包率 | 指标 | KCP | QCP | 提升 |
|--------|------|-----|-----|------|
| **0%** | P50 | 3.99ms | 1.79ms | **↓55%** |
| **0%** | P99 | 5.79ms | 2.71ms | **↓53%** |
| **5%** | P50 | 3.93ms | 1.73ms | **↓56%** |
| **5%** | P99 | 20.89ms | 4.66ms | **↓78%** |
| **10%** | P50 | 3.95ms | 1.73ms | **↓56%** |
| **10%** | P99 | 22.04ms | 4.80ms | **↓78%** |

### 完整对比表

#### 0% 丢包

```
Proto  │      P50 │      P95 │      P99 │ Throughput │ Bandwidth
───────┼──────────┼──────────┼──────────┼────────────┼──────────
TCP    │    276µs │  1.63ms  │  2.39ms  │  94.4 MB/s │     100%
UDP    │  1.68ms  │  2.16ms  │  2.54ms  │  28.6 MB/s │      80%
KCP    │  3.99ms  │  5.35ms  │  5.79ms  │  26.4 MB/s │     109%
QCP    │  1.79ms  │  2.31ms  │  2.71ms  │  27.2 MB/s │     104%

★ QCP vs KCP: P50 ↓55%  P99 ↓53%
```

#### 5% 丢包

```
Proto  │      P50 │      P95 │      P99 │ Throughput │ Bandwidth
───────┼──────────┼──────────┼──────────┼────────────┼──────────
TCP    │    323µs │  1.71ms  │  2.56ms  │  86.9 MB/s │     100%
UDP    │  1.72ms  │  2.21ms  │  2.56ms  │  28.0 MB/s │      80%
KCP    │  3.93ms  │ 10.43ms  │ 20.89ms  │  28.2 MB/s │     109%
QCP    │  1.73ms  │  2.63ms  │  4.66ms  │  28.4 MB/s │     104%

★ QCP vs KCP: P50 ↓56%  P99 ↓78%
```

#### 10% 丢包

```
Proto  │      P50 │      P95 │      P99 │ Throughput │ Bandwidth
───────┼──────────┼──────────┼──────────┼────────────┼──────────
TCP    │    257µs │  1.60ms  │  2.39ms  │  98.8 MB/s │     100%
UDP    │  1.69ms  │  2.22ms  │  2.63ms  │  28.3 MB/s │      80%
KCP    │  3.95ms  │ 17.49ms  │ 22.04ms  │  28.6 MB/s │     109%
QCP    │  1.73ms  │  3.64ms  │  4.80ms  │  28.6 MB/s │     104%

★ QCP vs KCP: P50 ↓56%  P99 ↓78%
```

## 为什么 QCP 碾压 KCP？

| 技术 | KCP | QCP 2026 | 影响 |
|------|-----|----------|------|
| 丢包恢复 | ARQ 重传 | FEC 前向纠错 | 丢包无需等待重传 |
| 丢包开销 | 8-20ms | 1-2μs (SIMD) | 降低 1000x |
| 拥塞控制 | TCP-like | AI 预测 | 延迟更稳定 |
| 内存分配 | 每包分配 | Zero-copy ring | GC 归零 |
| 锁竞争 | Mutex | Lock-free | 无等待 |
| 协议开销 | 24 bytes | 10 bytes | 带宽省 58% |

### QCP 核心创新

1. **FEC (Forward Error Correction)**
   - 自适应 Reed-Solomon 编码
   - 95% 丢包无需重传
   - SIMD 硬件加速解码

2. **AI 拥塞控制**
   - 预测网络状态
   - 主动调整发送速率
   - 避免拥塞发生

3. **Zero-Copy Ring Buffer**
   - 预分配内存池
   - 无 GC 压力
   - CPU cache 友好

4. **Lock-Free 数据结构**
   - 无 mutex 竞争
   - 无 goroutine 调度开销
   - 适合高并发

## License

MIT License

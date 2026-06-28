# QCP Benchmark

QCP vs KCP vs TCP vs UDP — 2026 颠覆性协议性能对比

## 核心结论

**QCP 不是 UDP 上的 TCP，是为游戏而生的全新协议。**

| 丢包率 | KCP P50 | QCP P50 | 提升 |
|--------|---------|---------|------|
| 0% | 97.5ms | 1.7ms | **↓98%** |
| 5% | 98.4ms | 1.7ms | **↓98%** |
| 10% | 99.7ms | 1.7ms | **↓98%** |

## 快速开始

```bash
docker-compose up -d server
docker-compose run --rm client -mode client -server 10.10.0.10:9000 -protocol all -duration 5s -connections 100 -loss 0.05
```

## 协议层创新

| 维度 | KCP | QCP 2026 |
|------|-----|----------|
| 可靠性机制 | ARQ (每包重传) | FEC (前向纠错) |
| 丢包处理 | 等超时+重传 | FEC实时解码 |
| 包头开销 | 24 bytes | 10 bytes |
| 队列模型 | 单一 FIFO | 三通道优先级 |
| 内存管理 | 每次分配 | Zero-Copy Ring |
| 锁竞争 | Mutex | Lock-Free |
| 拥塞控制 | TCP Reno | AI 预测 |
| 网络切换 | 断开重连 | 无缝迁移 |

## 实测结果

### 0% 丢包

```
Proto  │      P50 │      P95 │      P99 │ Throughput │ Bandwidth
───────┼──────────┼──────────┼──────────┼────────────┼──────────
TCP    │    277µs │  1.71ms  │  2.75ms  │  90.2 MB/s │     100%
UDP    │  1.91ms  │  3.01ms  │  4.68ms  │  23.7 MB/s │      80%
KCP    │ 97.46ms  │ 108.67ms │ 113.90ms │   0.5 MB/s │     109%
QCP    │  1.68ms  │  2.19ms  │  2.56ms  │  29.1 MB/s │     104%

★ QCP vs KCP: P50 ↓98%  P99 ↓98%
```

### 5% 丢包

```
Proto  │      P50 │      P95 │      P99 │ Throughput │ Bandwidth
───────┼──────────┼──────────┼──────────┼────────────┼──────────
TCP    │    281µs │  1.64ms  │  2.46ms  │  94.8 MB/s │     100%
UDP    │  1.66ms  │  2.13ms  │  2.48ms  │  28.9 MB/s │      80%
KCP    │ 98.41ms  │ 110.05ms │ 115.77ms │   0.5 MB/s │     109%
QCP    │  1.72ms  │  2.24ms  │  2.70ms  │  28.3 MB/s │     104%

★ QCP vs KCP: P50 ↓98%  P99 ↓98%
```

### 10% 丢包

```
Proto  │      P50 │      P95 │      P99 │ Throughput │ Bandwidth
───────┼──────────┼──────────┼──────────┼────────────┼──────────
TCP    │    219µs │  1.57ms  │  2.36ms  │ 104.4 MB/s │     100%
UDP    │  1.67ms  │  2.18ms  │  2.61ms  │  28.6 MB/s │      80%
KCP    │ 99.72ms  │ 112.99ms │ 120.41ms │   0.5 MB/s │     109%
QCP    │  1.72ms  │  2.30ms  │  2.89ms  │  28.5 MB/s │     104%

★ QCP vs KCP: P50 ↓98%  P99 ↓98%
```

## 为什么 QCP 碾压 KCP？

### KCP 的致命缺陷

```
KCP 流程:
发送包 → 丢包 → 等RTO超时(8-20ms) → 重传 → 等ACK → 收到
         ↓
    每个丢包都卡 8-20ms
```

### QCP 的颠覆

```
QCP 流程:
发送包+FEC冗余 → 丢包 → FEC解码恢复(1μs) → 完成
         ↓
    丢包几乎无感
```

### 技术栈对比

| 技术 | KCP | QCP 2026 |
|------|-----|----------|
| 丢包恢复 | ARQ重传 | FEC纠错 |
| 丢包开销 | 8-20ms | 1-2μs |
| 拥塞控制 | TCP Reno | AI预测 |
| 内存分配 | 每包分配 | Zero-Copy |
| 锁竞争 | Mutex | Lock-Free |
| 协议头 | 24 bytes | 10 bytes |
| 批量发送 | 不支持 | 10ms批次 |

## License

MIT License

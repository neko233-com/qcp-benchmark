# QCP Benchmark

**QCP vs KCP vs TCP vs UDP — 性能压测**

> KCP 代码在 `baseline_kcp.go` 中，**仅作压测对照基线**，不是 QCP 的组件或依赖。

## 核心结论

QCP 在 P50/P99 延迟上**完全吊打 KCP**（5G 模拟 RTT 下 Fast NACK vs KCP RTO）。

| 丢包率 | KCP P50 | QCP P50 | 提升 |
|--------|---------|---------|------|
| 0% | ~97ms | ~1.7ms | **↓98%** |
| 5% | ~98ms | ~1.7ms | **↓98%** |
| 10% | ~100ms | ~1.7ms | **↓98%** |

## 快速开始

```bash
go run . -mode all -duration 5s -loss 0.05
```

Docker:

```bash
docker-compose up -d server
docker-compose run --rm client -mode client -server 10.10.0.10:9000 -protocol all -duration 5s
```

## 文件说明

| 文件 | 说明 |
|------|------|
| `main.go` | QCP / TCP / UDP 压测 |
| `baseline_kcp.go` | **KCP 基线对手（测试专用）** |

## QCP vs KCP 架构差异

| 维度 | KCP (基线) | QCP |
|------|-----------|-----|
| 丢包恢复 | RTO 8-20ms | Fast NACK ~5ms |
| 状态数据 | 可靠重传 | REALTIME 最新覆盖 |
| 场景 | 通用 | 游戏 / IoT |
| 并发 | ~10K | 100K+ Lock-Free |

详见 [docs/BASELINE.md](../docs/BASELINE.md)

## License

MIT License

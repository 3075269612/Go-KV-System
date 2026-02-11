# Flux-KV Performance Benchmark Report (Day 26)

## 1. 测试环境 (Test Environment)
- **OS**: Linux
- **Language**: Go (GMP Scheduling)
- **Architecture**: Microservices (gRPC + Etcd + RabbitMQ)
- **Client**: Custom Benchmark Tool (`cmd/benchmark`)
- **Server**: Flux-KV Solo Node (ShardedMap Enabled)
- **Metric**: Write-Heavy Scenario (100% Set operations)

## 2. 也是测试目标 (Test Objectives)
1. **CDC 性能影响分析 (CDC Impact Analysis)**: 验证开启 Change Data Capture (异步写入 RabbitMQ) 对主流程写入性能的影响。
2. **锁竞争分析 (Lock Contention)**: 通过 Pprof 验证 ShardedMap 在高并发下的表现。

## 3. 测试结果 (Benchmark Results)

### 3.1 核心指标对比 (Key Metrics)

| 场景 (Scenario) |并发数 (Concurrency)| 总请求 (Total Requests) | 平均 QPS (Req/s) | 耗时 (Duration) | 成功率 |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **CDC Disabled** | 100 | 500,000 | **42,285** | 11.82s | 100% |
| **CDC Enabled** | 100 | 500,000 | **32,340** | 15.46s | 100% |

> **结论**: 开启 CDC 后，写入性能下降约 **23.5%** (42k -> 32k)。
> **分析**: 性能损耗主要源于 RabbitMQ 的网络 I/O 以及数据序列化（JSON Marshal）开销。虽然采用了异步 `EventBus`，但在超高并发下，Channel 的锁竞争和 Goroutine 调度仍有一定成本。但在业务上，这换取了数据的一致性流转，是可以接受的 Trade-off。

### 3.2 Pprof 性能分析 (Profiling)

本次测试生成了两个 CPU Profile 文件：
- `cpu_cdc_off.prof`: 未开启 CDC
- `cpu_cdc_on.prof`: 开启 CDC

#### 分析发现 (Observations):
1. **ShardedMap 效果显著**: 在 Profile 图中，`sync.Mutex` 或 `RWMutex` 的等待时间并没有出现在 Top 耗时中（或者占比很低）。这证明 `ShardedMap` (256 分片) 极大地稀释了锁竞争，让多核 CPU 能够跑满业务逻辑。
2. **Syscall 开销**: 大量的 CPU 时间消耗在 `syscall.Write` (网络 IO) 和 `runtime.mallocgc` (内存分配) 上，这是高吞吐系统的正常表现。

## 4. 复现步骤 (How to Reproduce)

### 4.1 启动基础组件
```bash
docker-compose up -d etcd rabbitmq jaeger
```

### 4.2 编译
```bash
go build -o bin/flux-server cmd/server/main.go
go build -o bin/benchmark cmd/benchmark/main.go
```

### 4.3 运行 Server (CDC Enabled)
```bash
export FLUX_PPROF_ENABLED=true
export FLUX_RABBITMQ_URL="amqp://fluxadmin:flux2026secure@localhost:5672/"
export FLUX_POD_IP=127.0.0.1
./bin/flux-server
```

### 4.4 运行压测
```bash
./bin/benchmark -c 100 -n 500000
```

### 4.5 查看 Pprof
```bash
go tool pprof -http=:8080 cpu_cdc_on.prof
```

## 5. 面试话术 (Interview Talking Points)
- "在压测中，我发现开启 CDC 对写性能有约 20% 的损耗。为了解决这个问题，我最初考虑过完全异步（Fire-and-Forget），但这可能导致消息丢失。目前的方案是在 Handler 层直接投递到带缓冲的 Channel，平衡了性能和可靠性。"
- "通过 Pprof 分析，我确认了 `internal/core/mem_db.go` 中的 `ShardedMap` 实现有效地避免了全局锁热点，Profile 中锁等待几乎不可见。"

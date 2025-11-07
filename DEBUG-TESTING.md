# Debug Testing Guide

## 调试配置说明

已创建 `.env.debug` 配置文件，用于调试模式测试。

### 调试配置特点

1. **不同端口**: 使用 `:9999` 端口，避免与生产环境 (`:8480`) 冲突
2. **禁用数据采集**:
   - `VM_ENABLE_SHORT=false` - 不写入 10 秒数据
   - `VM_ENABLE_LONG=false` - 不写入 5 分钟数据
3. **启用查询**: `VM_ENABLED=true` - 可以查询 VictoriaMetrics 中的历史数据
4. **单接口监控**: 只监控 `vlan2622`，减少资源占用

## 使用方法

### 方法 1: 使用批处理脚本

```cmd
test-debug.bat
```

这会启动调试实例在端口 9999。

### 方法 2: 命令行直接启动

```cmd
mikrotik-stats.exe --env=.env.debug
```

### 方法 3: 自定义配置文件

```cmd
mikrotik-stats.exe --env=my-custom.env
```

## 测试查询

### 使用 PowerShell 脚本测试

```powershell
.\test-query.ps1
```

这会测试查询最近 24 小时的数据，使用 30 分钟间隔。

### 使用浏览器测试

1. 启动调试实例（端口 9999）
2. 打开浏览器访问: http://localhost:9999/history.html?interface=vlan2622
3. 选择时间范围和间隔，点击查询

### 使用 curl 测试

```bash
curl "http://localhost:9999/api/history?interface=vlan2622&start=1730872801&end=1730959201&interval=30m"
```

## 调试日志说明

程序会输出详细的调试信息，以 `[VM]` 开头：

```
[Config] Loading configuration from: .env.debug
[VM] VictoriaMetrics client initialized (URL: http://43.161.250.190:28428)
[Web] Starting web server on :9999
[VM] Querying history: interface=vlan2622, interval=30m (converted to 1800s), range=14:20:01 to 14:20:01
[VM] Executing query for upload_avg: mikrotik_interface_tx_rate_avg{interface="vlan2622",interval="1800s"}
[VM] Full request URL: http://43.161.250.190:28428/api/v1/query_range?query=...&start=...&end=...&step=60
[VM] VM Response status: success, result count: 1
[VM] First result has 48 values, metric labels: map[__name__:mikrotik_interface_tx_rate_avg interface:vlan2622 interval:1800s]
[VM] Query upload_avg returned 48 data points
```

### 关键日志解读

1. **interval 转换**:
   ```
   interval=30m (converted to 1800s)
   ```
   确认前端的友好格式（30m）正确转换为 VictoriaMetrics 存储格式（1800s）

2. **查询语句**:
   ```
   mikrotik_interface_tx_rate_avg{interface="vlan2622",interval="1800s"}
   ```
   确认使用正确的 label 格式查询

3. **响应结果**:
   ```
   result count: 1
   First result has 48 values
   ```
   如果 result count = 0，说明 VM 中没有匹配的数据

4. **数据点数量**:
   ```
   Query upload_avg returned 48 data points
   ```
   48 个数据点 = 24 小时 / 30 分钟间隔，符合预期

## 常见问题

### 查询返回空数据

如果看到：
```
[VM] WARNING: Query returned 0 results. This means no data matched the query.
```

可能的原因：

1. **Interval 不匹配**:
   - VictoriaMetrics 中只有 `10s` 和 `300s` 的数据
   - 前端查询的是 `1800s` (30 分钟)
   - **解决**: 前端选择 10秒 或 5分钟 间隔

2. **Interface 名称不匹配**:
   - 检查 VictoriaMetrics 中实际存储的接口名称

3. **时间范围没有数据**:
   - 检查选择的时间范围内是否有数据采集

### 端口被占用

如果 9999 端口被占用，修改 `.env.debug` 中的：
```
WEB_LISTEN_ADDR=:9998
```

## 生产环境注意事项

调试完成后：

1. **停止调试实例**: 按 Ctrl+C
2. **不要提交调试日志**: `.env.debug` 可以提交，但不要提交含敏感信息的日志
3. **恢复生产配置**: 生产环境继续使用 `.env` 文件

# Mikrotik 接口流量监控

**专业的 Mikrotik 路由器实时网络流量监控工具**

通过现代化的 Web 仪表板、终端输出和基于 VictoriaMetrics 的历史数据分析，监控 Mikrotik 接口流量。

![Web 仪表板](docs/screenshot.png)
*实时监控，支持响应式 1-4 列网格布局*

[English](README.md) | 简体中文

## 功能特性

### 核心监控
- ✅ 使用 MD5 质询-响应认证连接到 Mikrotik API
- ✅ 通过 .env 配置可监控接口列表
- ✅ 精确到秒的流量速率计算
- ✅ **用户友好的上传/下载显示**（自动处理上行/下行接口）
- ✅ 多种终端显示模式（refresh/append/log）
- ✅ 可配置的速率单位（比特/字节每秒）
- ✅ 自动缩放或固定比例显示，带小数对齐
- ✅ **性能优化**：条件性统计计算（仅在需要时计算）

### Web 界面（现代化 & 专业）
- ✅ **实时监控**，基于 WebSocket（1 秒更新）
- ✅ **响应式 1-4 列网格**布局（自适应屏幕尺寸）
- ✅ **交互式 Chart.js 图表**，60 秒滚动窗口
- ✅ **可折叠统计面板**，显示当前速度、平均值和峰值
- ✅ **前端计算统计**（10 秒滚动窗口）
- ✅ **模态框图表放大**，用于详细分析
- ✅ **接口标签系统**，支持自定义名称
- ✅ **简洁现代的暗色主题**，优化监控体验
- ✅ **历史数据查询**界面，支持时间范围选择
- ✅ **嵌入式静态文件**（单文件分发，支持开发模式热重载）

### 数据管理
- ✅ **VictoriaMetrics 集成**，用于历史数据存储
- ✅ **双间隔聚合**（10 秒短期，5 分钟长期）
- ✅ **基于 PromQL 的查询**，自动选择间隔
- ✅ **优化数据传输**（WebSocket 负载减少 67%）
- ✅ **自动重连**，网络中断时

## 配置

在项目根目录创建 `.env` 文件或设置环境变量：

```env
MIKROTIK_HOST=175.100.109.154
MIKROTIK_PORT=65428
MIKROTIK_USERNAME=your_username
MIKROTIK_PASSWORD=your_password

# 接口列表（逗号分隔）
INTERFACES=vlan2622,vlan2624

# 上行接口（可选，逗号分隔）
UPLINK_INTERFACES=

# 显示模式（可选）
DISPLAY_MODE=refresh  # 或 "append"

# 速率单位（可选）
RATE_UNIT=auto  # 或 "bps" (比特/秒), "Bps" (字节/秒)

# 速率比例（可选）
RATE_SCALE=auto  # 或 "k", "M", "G" 固定比例

# 输出模式（可选）
OUTPUT_MODE=terminal  # 或 "log"

# 调试模式（可选）
DEBUG=false  # 或 "true" 查看 API 命令

# Web 界面（可选）
WEB_ENABLED=true
WEB_LISTEN_ADDR=:8080

# VictoriaMetrics（可选）
VM_ENABLED=true
VM_URL=http://localhost:8428
VM_SHORT_INTERVAL=10s
VM_LONG_INTERVAL=300s
VM_ENABLE_SHORT=true
VM_ENABLE_LONG=true
```

**配置选项说明：**

- **INTERFACES**: 要监控的接口列表，逗号分隔（默认：vlan2622,vlan2624）

- **UPLINK_INTERFACES**: 上行接口列表，逗号分隔（可选）
  - **上行（WAN 到 ISP）**: TX=上传，RX=下载（正常理解）
  - **下行（LAN/VLAN 到用户）**: TX=下载（到用户），RX=上传（来自用户）- 需要交换
  - 示例：`UPLINK_INTERFACES=ether1,sfp1` 如果 ether1 和 sfp1 连接到 ISP
  - 如果所有监控接口都是下行（如 LAN、VLAN），则留空
  - **为什么需要？** 对于下行接口，路由器发送数据到用户（TX），这实际上是用户的下载。路由器接收来自用户的数据（RX），这实际上是用户的上传。

- **DISPLAY_MODE**: 输出显示方式
  - `refresh`（默认）- 像 `top`/`htop` 一样重绘显示
    - 使用 ANSI 光标控制（移动到起始位置并覆盖）
    - 不进行全屏清除，减少闪烁
    - **推荐终端：**
      - Windows: Windows Terminal, PowerShell 7+, Git Bash
      - Linux/macOS: 任何标准终端
  - `append` - 像 `tail -f` 一样追加新行
    - 适合日志记录和重定向到文件

- **RATE_UNIT**: 流量速率显示单位
  - `auto`（默认）- 使用字节每秒（B/s）
  - `bps` - 比特每秒（乘以 8）
  - `Bps` - 字节每秒

- **RATE_SCALE**: 速率显示比例
  - `auto`（默认）- 自动缩放（B/s, KB/s, MB/s, GB/s）
  - `k` - 固定千比特/千字节比例
  - `M` - 固定兆比特/兆字节比例（适合高速接口）
  - `G` - 固定吉比特/吉字节比例
  - 固定比例使用 7.2f 格式进行小数对齐（例如，"  12.34 Mbps"）

- **OUTPUT_MODE**: 输出格式
  - `terminal`（默认）- 格式化表格输出，用于交互使用
  - `log` - 日志风格输出，用于作为服务/守护进程运行

- **DEBUG**: 启用调试输出
  - `false`（默认）- 正常运行
  - `true` 或 `1` - 打印正在发送的 Mikrotik API 命令（用于故障排查）

- **WEB_ENABLED**: 启用 Web 界面
  - `true` - 启用带有 Chart.js 图表的实时 Web 仪表板
  - `false`（默认）- 禁用 Web 界面

- **WEB_LISTEN_ADDR**: Web 服务器监听地址
  - 默认：`:8080` - 监听所有接口，端口 8080
  - 示例：`localhost:8080` - 仅从本地访问

- **VM_ENABLED**: 启用 VictoriaMetrics 集成
  - `true` - 启用指标聚合并推送到 VictoriaMetrics
  - `false`（默认）- 禁用 VictoriaMetrics 集成

- **VM_URL**: VictoriaMetrics 服务器 URL
  - 默认：`http://localhost:8428`
  - 必须包含协议（http/https）

- **VM_SHORT_INTERVAL**: 短期聚合间隔
  - 默认：`10s` - 10 秒窗口，用于详细监控
  - 适合 <1 小时时间范围

- **VM_LONG_INTERVAL**: 长期聚合间隔
  - 默认：`300s`（5 分钟）- 5 分钟窗口，用于历史数据
  - 适合 >1 小时时间范围

- **VM_ENABLE_SHORT**: 启用短期聚合
  - `true`（默认）- 启用 10 秒聚合
  - `false` - 禁用短期指标

- **VM_ENABLE_LONG**: 启用长期聚合
  - `true`（默认）- 启用 5 分钟聚合
  - `false` - 禁用长期指标

参考 `.env.example`。

### Windows 终端支持

**好消息：** 程序会自动在 Windows 上启用虚拟终端处理！

这意味着刷新模式可以在以下环境工作：
- ✅ Windows CMD（自动启用）
- ✅ Windows Terminal
- ✅ PowerShell（所有版本）
- ✅ Git Bash
- ✅ 任何 Windows 控制台

无需手动配置！程序在启动时使用 Windows API 启用 ANSI 支持。

如果您仍然看到像 `[2J[H]` 这样的转义码（非常罕见），可以切换到追加模式：
```bash
DISPLAY_MODE=append
```

## 使用方法

不构建直接运行（推荐用于开发）：
```bash
go run .
```

构建并运行：
```bash
go build -o mikrotik-stats
./mikrotik-stats
```

### Web 界面

当 Web 界面启用（`WEB_ENABLED=true`）时，访问仪表板：
```
http://localhost:8080
```

**主仪表板：**
- ✅ **响应式网格布局**：根据屏幕宽度自动调整 1 到 4 列
  - ≥1920px: 4 列（完美适配 4K 显示器）
  - 1400-1919px: 3 列（标准桌面）
  - 900-1399px: 2 列（笔记本）
  - <900px: 1 列（移动/平板）
- ✅ **实时 Chart.js 图表**：60 秒滚动窗口，带上传/下载线
- ✅ **可折叠统计**：点击展开查看 10 秒平均值和峰值
- ✅ **模态框放大视图**：点击图表打开全屏详细分析
- ✅ **自定义接口标签**：编辑接口名称以便更容易识别
- ✅ **WebSocket 实时更新**：亚秒级延迟，自动重连
- ✅ **简洁界面**：无边框，透明卡片，优化高密度监控

**历史数据：**
点击任何接口卡片上的 "📊 历史" 按钮：
- 从 VictoriaMetrics 查看历史数据
- 选择时间范围（1 小时到 30 天）
- 自动间隔选择（<1h 为 10 秒，>1h 为 5 分钟）
- 查看由 VictoriaMetrics PromQL 计算的 4 个指标：
  - 上传/下载平均值（平均速率）
  - 上传/下载峰值（最大速率）
- 交互式 Chart.js 图表，带时间轴
- 可导出的数据可视化

**设置：**
通过页眉中的 ⚙️ 图标访问：
- 自定义接口显示名称
- 更改保存到服务器配置
- 在所有连接的客户端之间同步

**开发者模式：**
- 如果 `web/` 目录存在：使用本地文件（开发热重载）
- 否则：使用二进制嵌入的文件（生产）
- 无需单独的静态文件分发
- 单个可执行文件部署

## 输出示例

**刷新模式（默认），7 列数字显示：**
```
Mikrotik 接口流量监控
================================================================================
时间: 2025-11-07 01:08:36
单位: MB/s
--------------------------------------------------------------------------------
接口             上传      下载      上传平均   下载平均   上传峰值   下载峰值
--------------------------------------------------------------------------------
vlan2622         0.03       0.03       0.03       0.03       0.04       0.03
vlan2624         0.58       3.19       0.52       3.10       0.65       3.50
--------------------------------------------------------------------------------
按 Ctrl+C 停止
```

**特性：**
- **顶部单位**：单一单位显示（如 "MB/s"、"kbps"）适用于所有列
- **纯数字**：所有值都是带 .00 小数格式的数字
- **右对齐**：便于视觉比较值
- **实时速率**：当前上传/下载速度（Up/Down）
- **10 秒平均值**：UpAvg/DnAvg - 过去 10 秒的平滑速率
- **10 秒峰值**：UpPeak/DnPeak - 过去 10 秒的最大速度
- **80 列显示**：7 列 × 10 字符 = 70 字符（适合标准终端）

注意：显示从用户角度显示 "上传" 和 "下载"。如果接口配置为上行，RX/TX 会自动交换。

**追加模式：**
```
2025/11/07 01:09:12 已连接到 Mikrotik：175.100.109.154:65428

监控接口流量（Ctrl+C 停止）：
================================================================================
[2025-11-07 01:09:13] vlan2622: 上传: 18.12 KB/s  下载: 15.82 KB/s
[2025-11-07 01:09:13] vlan2624: 上传: 655.47 KB/s  下载: 3.00 MB/s
[2025-11-07 01:09:14] vlan2622: 上传: 50.48 KB/s  下载: 16.60 KB/s
[2025-11-07 01:09:14] vlan2624: 上传: 431.54 KB/s  下载: 3.64 MB/s
```

**固定比例模式（RATE_SCALE=M）：**
```
Mikrotik 接口流量监控
================================================================================
时间: 2025-11-07 01:08:36
--------------------------------------------------------------------------------
接口            上传                 下载
--------------------------------------------------------------------------------
vlan2622          0.03 MB/s            0.03 MB/s
vlan2624          0.58 MB/s            3.19 MB/s
--------------------------------------------------------------------------------
按 Ctrl+C 停止
```

**日志模式（OUTPUT_MODE=log）：**
```
2025/11/07 01:09:12 已连接到 Mikrotik：175.100.109.154:65428
2025/11/07 01:09:12 Mikrotik 接口流量监控已启动
2025/11/07 01:09:13 interface=vlan2622 upload=18.12 KB/s download=15.82 KB/s
2025/11/07 01:09:13 interface=vlan2624 upload=655.47 KB/s download=3.00 MB/s
2025/11/07 01:09:14 interface=vlan2622 upload=50.48 KB/s download=16.60 KB/s
2025/11/07 01:09:14 interface=vlan2624 upload=431.54 KB/s download=3.64 MB/s
```

## 要求

- Go 1.21 或更高版本
- 访问启用了 API 的 Mikrotik 路由器
- 有效的 Mikrotik 凭据

## 项目结构

```
.
├── main.go                 # 程序入口
├── config.go               # 配置加载
├── client.go               # Mikrotik API 客户端
├── stats.go                # 统计数据结构和格式化
├── monitor.go              # 监控逻辑
├── output.go               # 输出抽象（终端/日志模式）
├── web.go                  # Web 服务器，带 WebSocket + 嵌入式文件
├── vm.go                   # VictoriaMetrics 客户端和聚合
├── terminal_windows.go     # Windows ANSI 支持（构建标签：windows）
├── terminal_unix.go        # Unix ANSI 存根（构建标签：!windows）
├── web/                    # Web 界面文件（嵌入式）
│   ├── index.html          # 主 HTML 结构
│   └── static/
│       ├── css/
│       │   └── style.css   # 暗色主题样式
│       └── js/
│           └── app.js      # 实时 + 历史数据逻辑
├── go.mod                  # Go 模块配置
├── .env                    # 环境配置
├── DEPLOYMENT.md           # 部署指南
├── README.md               # 英文文档
└── README.zh-CN.md         # 中文文档
```

## 架构亮点

本项目展示了现代 Go 实践和高效的数据流设计：

### 关注点分离

**数据层（后端）**：
- 从 Mikrotik API 收集原始流量速率
- 仅在需要时计算统计信息（终端/日志输出）
- 将历史数据存储到 VictoriaMetrics
- 通过 WebSocket 发送最小负载（仅瞬时速率）

**视图层（前端）**：
- 从实时数据计算自己的统计信息
- 独立的 10 秒滑动窗口
- 灵活的窗口大小，无需后端更改
- 响应式 UI，自动列调整

**存储层（VictoriaMetrics）**：
- 使用 PromQL 进行服务器端聚合
- 双间隔策略（10s + 5min）
- 基于查询范围自动选择间隔

### 性能优化

1. **条件性统计计算**（`monitor.go:152`）
   - 仅在启用终端或日志输出时计算 avg/peak
   - 仅 Web 模式时跳过环形缓冲区更新
   - 为高频监控节省 CPU 周期

2. **WebSocket 负载减少**（`web.go:299-302`）
   - 移除 4 个冗余字段（avg/peak/mbps）
   - 数据大小减少 67%（6 个字段 → 2 个字段）
   - 更低带宽，更快传输

3. **前端独立性**（`app.js:53-103`）
   - 自包含的统计计算
   - 不依赖后端计算的值
   - 可以在客户端调整窗口大小

4. **响应式网格布局**（`style.css:122-151`）
   - CSS Grid，带 auto-fit 和媒体查询
   - 适应屏幕宽度：1/2/3/4 列
   - 优化 4K 显示器和移动设备

### 代码质量

- ✅ **生产环境无 console.log**：所有调试代码已移除
- ✅ **无未使用的 CSS**：清理了 69 行遗留样式
- ✅ **无死代码**：通过 `go vet`，零警告
- ✅ **清晰分离**：数据/视图/存储层独立
- ✅ **单一二进制**：嵌入式文件，无外部依赖

## 实现细节

### 核心监控
- 直接使用 Mikrotik API 协议（无外部依赖）
- 实现正确的 MD5 质询-响应认证
- 使用 Mikrotik API 查询语法进行服务器端过滤（减少网络开销）
- 存储先前的字节计数以计算每秒增量
- 使用 `time.Ticker` 实现精确的 1 秒间隔
- 通过环境变量配置接口列表
- **在 Windows 上自动启用 ANSI 支持**，通过 Windows API（无需手动设置）
- **性能优化**：仅在启用终端/日志输出时计算统计信息

### 输出系统
- 具有 OutputWriter 接口的模块化输出系统：
  - TerminalOutput: 交互式显示（refresh/append 模式）
  - LogOutput: 服务友好的结构化日志
  - WebServer: 实时 WebSocket 仪表板
- 可配置的速率单位（比特 vs 字节）和比例（自动/固定）
- 固定比例格式化，带小数对齐，便于阅读
- **高效的光标控制**：使用 ANSI 转义序列移动光标，而不是清除屏幕
  - 减少闪烁，提高视觉稳定性
  - 接口始终按字母顺序显示（不跳动）

### Web 界面架构
- **嵌入式静态文件**：Go 1.16+ `//go:embed` 指令
- **开发者模式**：自动检测 `web/` 目录以进行热重载
- **生产模式**：使用二进制嵌入的文件（单个可执行文件）
- **WebSocket 优化**：最小的 JSON 负载（仅 upload_rate/download_rate）
  - 相比以前版本减少 67% 的数据大小
  - 前端独立计算统计信息
- **Chart.js 4.4.0**：现代、响应式图表，带时间轴
- **暗色主题**：优化监控的 Slate 配色方案
- **前端统计**：JavaScript 中的 10 秒滑动窗口计算
  - 消除后端到前端的依赖
  - 无需后端重启即可灵活调整窗口大小

### VictoriaMetrics 集成
- **固定时间边界聚合**：窗口对齐到间隔（非滑动）
- **双间隔支持**：10s（短期）+ 300s（长期）
- **Prometheus 格式**：与标准 VM 导入 API 兼容
- **重试逻辑**：自动重试，带指数退避
- **查询 API**：基于 PromQL 的历史数据检索，自动聚合
- **自动间隔选择**：根据时间范围选择适当的粒度
- **服务器端计算**：使用 VictoriaMetrics PromQL 函数进行精确统计
  - `rate()` 用于平均值计算
  - `max_over_time()` 用于峰值检测
- **导出的指标**：
  - `mikrotik_interface_rx_rate_avg{interface,interval}` - 平均下载速率
  - `mikrotik_interface_rx_rate_peak{interface,interval}` - 峰值下载速率
  - `mikrotik_interface_rx_rate_min{interface,interval}` - 最小下载速率
  - `mikrotik_interface_tx_rate_avg{interface,interval}` - 平均上传速率
  - `mikrotik_interface_tx_rate_peak{interface,interval}` - 峰值上传速率
  - `mikrotik_interface_tx_rate_min{interface,interval}` - 最小上传速率
  - `mikrotik_interface_sample_count{interface,interval}` - 样本数量

## API 查询格式

程序使用以下 Mikrotik API 命令格式：
```
/interface/print
=stats
=.proplist=name,rx-byte,tx-byte
?name=vlan2622
?name=vlan2624
?#|
?name=vlan2626
?#|
```

**说明：**
- `=stats` - 获取实时统计信息（活动计数器）
- `=.proplist=name,rx-byte,tx-byte` - 仅返回这些属性
- `?name=<interface>` - 按接口名称过滤
- `?#|` - OR 运算符，**从第二个接口开始放在每个接口之后**

**重要提示：** OR 运算符 `?#|` 必须**从第二个接口开始放在每个接口之后**。这允许查询匹配 interface1 或 interface2 或 interface3，等等。

格式模式：
- 1 个接口：`?name=iface1`
- 2 个接口：`?name=iface1 ?name=iface2 ?#|`
- 3 个接口：`?name=iface1 ?name=iface2 ?#| ?name=iface3 ?#|`

这会在 Mikrotik 路由器上过滤结果后再发送，减少网络流量和处理时间。

**故障排查：** 如果遇到多个接口的问题，在 .env 文件中设置 `DEBUG=true` 启用调试模式。这将打印正在发送的实际 API 命令，以帮助诊断问题。

## 许可证

MIT

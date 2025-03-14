# ComServer

ComServer 是一个串口通信服务器，用于在串口设备和网络之间转发数据，支持流控信号的处理。它可以让您通过网络远程访问和控制串口设备，特别适合用于远程调试和刷写 Arduino、ESP32 等开发板。

## 使用场景

- 远程调试 Arduino、ESP32 等开发板
- 远程刷写固件
- 远程串口调试
- 串口设备的远程控制

## 特点

- 支持串口和网络之间的双向数据转发
- 支持流控信号（CTS、DCD、DSR、RI）的处理
- 可配置的流控信号反转
- 支持大包分片传输
- 实时状态监控和日志记录

## 配置

配置文件使用 YAML 格式，示例配置如下：

```yaml
server:
  port: 8080
  host: "0.0.0.0"

serial:
  address: "COM1"           # 串口地址
  baudrate: 115200         # 波特率
  databits: 8              # 数据位
  stopbits: 1              # 停止位
  parity: "N"              # 校验位 (N: None, E: Even, O: Odd)
  invert_cts: false        # 是否反转CTS信号
  invert_dcd: true         # 是否反转DCD信号
```

### 配置说明

- `server.port`: 服务器监听端口
- `server.host`: 服务器监听地址
- `serial.address`: 串口设备地址（Windows 下如 "COM1"，Linux 下如 "/dev/ttyUSB0"）
- `serial.baudrate`: 通信波特率
- `serial.databits`: 数据位（5-8）
- `serial.stopbits`: 停止位（1-2）
- `serial.parity`: 校验位（N: 无校验, E: 偶校验, O: 奇校验）
- `serial.invert_cts`: 是否反转CTS信号
- `serial.invert_dcd`: 是否反转DCD信号

## 数据包格式

### 网络数据包

1. 长度字节（1字节）：表示后续数据的总长度
2. 类型字节（1字节）：表示数据包类型
3. 数据部分（变长）：实际数据内容

### 数据包类型

- `0x01`: 数据包
- `0x02`: 流控状态包

### 流控状态字节

- Bit 0 (0x01): CTS (Clear To Send)
- Bit 1 (0x02): DSR (Data Set Ready)
- Bit 2 (0x04): DCD (Data Carrier Detect)
- Bit 3 (0x08): RI (Ring Indicator)

## 使用方法

1. 编译程序：
```bash
go build -o comserver.exe cmd/main.go
```

2. 运行程序：
```bash
comserver.exe -config config.yaml
```

## 注意事项

1. 确保有足够的串口访问权限
2. 流控信号的反转设置需要根据实际硬件连接方式配置
3. 大包传输会自动分片，每片最大255字节
4. 程序会实时监控并记录流控状态变化
5. 在 Windows 系统下，串口地址格式为 "COM1"、"COM2" 等
6. 在 Linux 系统下，串口地址格式为 "/dev/ttyUSB0"、"/dev/ttyACM0" 等
7. 刷写 ESP32 等设备时，请确保正确配置流控信号的反转设置
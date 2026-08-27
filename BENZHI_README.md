基于 Go 实现的 shadow 项目，一款后端服务，完成设备影子维护、命令下发与 OTA 分组路由。

shadow 是物联网设备影子与命令网关：设备长连接上报状态，平台维护 desired/reported 双态影子，控制面通过 HTTP 接口下发命令（含离线缓冲），并支持 OTA 版本与设备分组迁移、遥测接入与保留清理。

## 构建

```bash
go build -mod=vendor ./...
```

## 运行

```bash
go run -mod=vendor ./cmd/shadowd -config config.json
```

默认监听 `:9090`，健康检查为 `GET /healthz`。

## 主要接口

- `POST /api/v1/devices` 注册设备
- `GET /api/v1/devices/{id}/shadow` 查询影子
- `PUT /api/v1/devices/{id}/desired` 更新期望状态
- `POST /api/v1/devices/{id}/commands` 下发命令
- `POST /api/v1/devices/batch/desired` 批量下发期望状态
- `POST /api/v1/devices/{id}/move` 迁移 OTA 分组

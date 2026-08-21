# go-smtprelay

本地 SMTP 出站中继库：信封入队、MIME 组装、可选 DKIM 签名、按 MX/直连投递、bounce 分类与重试退避。

## 构建 / 测试

```text
go build ./...
go test ./... -count=1
go run ./cmd/maild -addr :8110
```

管理页默认监听 `:8110`，可查看队列、试提交与最近 bounce。

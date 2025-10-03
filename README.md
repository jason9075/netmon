# netmon (Vibe Coding)

每 10 秒對指定 URL 發送一次 HTTP GET；若逾時 > 2 秒或狀態碼非 2xx/3xx 視為失敗。日誌以 CSV-like 形式追加至 `/data/netlog.log`，可長期收集（例如 3 天）。

## 使用

```bash
make run   # 建置並以 Docker Compose 背景啟動
make logs  # 觀察即時輸出
make stop  # 關閉容器
```

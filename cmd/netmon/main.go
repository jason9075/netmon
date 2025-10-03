package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "strings"
    "syscall"
    "time"
)

func getenv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}

func atoienv(key string, def int) int {
    v := getenv(key, "")
    if v == "" {
        return def
    }
    if n, err := strconv.Atoi(v); err == nil {
        return n
    }
    return def
}

func main() {
    url := getenv("URL", "https://www.google.com/generate_204")
    intervalSec := atoienv("INTERVAL_SECONDS", 10)
    timeoutSec := atoienv("TIMEOUT_SECONDS", 2)
    logPath := getenv("LOG_PATH", "/data/netlog.log")

    // 建立 log 輸出到 stdout 與檔案
    if err := os.MkdirAll(strings.TrimSuffix(logPath, "/netlog.log"), 0o755); err != nil {
        // 即便失敗也不致命，後續打開檔案仍可能成功
        log.Printf("mkdir data dir failed: %v", err)
    }

    file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
    if err != nil {
        log.Fatalf("open log file failed: %v", err)
    }
    defer file.Close()

    writer := bufio.NewWriter(file)
    logger := log.New(os.Stdout, "", log.LstdFlags)

    client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}

    ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
    defer ticker.Stop()

    // OS 訊號（Ctrl+C / docker stop）
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    // 寫入檔頭（若新檔案）
    fi, _ := file.Stat()
    if fi != nil && fi.Size() == 0 {
        header := "# time_iso8601,ok,latency_ms,error\n"
        writer.WriteString(header)
        writer.Flush()
    }

    logger.Printf("Starting netmon: url=%s interval=%ds timeout=%ds log=%s", url, intervalSec, timeoutSec, logPath)

    for {
        select {
        case <-ctx.Done():
            logger.Println("Shutting down...")
            writer.Flush()
            return
        case t := <-ticker.C:
            started := time.Now()
            reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
            req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
            resp, err := client.Do(req)
            latency := time.Since(started)
            cancel()

            ok := false
            var statusCode int

            if err == nil {
                statusCode = resp.StatusCode
                // 視 2xx/3xx 為成功
                if statusCode >= 200 && statusCode < 400 {
                    ok = true
                }
                if resp.Body != nil {
                    // 讀掉並丟棄（避免連線重用受影響）
                    _ = resp.Body.Close()
                }
            }

            // 如果逾時，err 會是 context deadline exceeded 或 net error；統一視為失敗
            // 記錄一行 CSV-like 紀錄
            line := fmt.Sprintf("%s,%t,%d,%v\n", t.UTC().Format(time.RFC3339), ok, latency.Milliseconds(), err)
            if _, werr := writer.WriteString(line); werr != nil {
                logger.Printf("write log failed: %v", werr)
            }
            writer.Flush()

            if ok {
                logger.Printf("OK %d in %dms", statusCode, latency.Milliseconds())
            } else {
                logger.Printf("FAIL in %dms err=%v", latency.Milliseconds(), err)
            }
        }
    }
}

# go-redis-sniffer
go redis sniffer

```
$ ./go-redis-sniffer
sidecar for redis network monitoring, usage:
./go-redis-sniffer -port=6379
  -device string
        eth0 lo ... (default "any")
  -port int
        redis port
  -strict-mode
        error exit        
```

```
$ ./go-redis-sniffer -port=6379
[REDIS] 2025/06/09 15:11:41 127.0.0.1:53168 -> 127.0.0.1:6379 | PING
[REDIS] 2025/06/09 15:11:41 127.0.0.1:6379 <- 127.0.0.1:53168 | +PONG
[REDIS] 2025/06/09 15:11:41 127.0.0.1:53168 -> 127.0.0.1:6379 | SET hello world
[REDIS] 2025/06/09 15:11:41 127.0.0.1:6379 <- 127.0.0.1:53168 | +OK
[REDIS] 2025/06/09 15:11:41 127.0.0.1:53168 -> 127.0.0.1:6379 | GET hello
[REDIS] 2025/06/09 15:11:41 127.0.0.1:6379 <- 127.0.0.1:53168 | $5 world
[REDIS] 2025/06/09 15:11:41 127.0.0.1:53168 -> 127.0.0.1:6379 | QUIT
[REDIS] 2025/06/09 15:11:41 127.0.0.1:6379 <- 127.0.0.1:53168 | +OK
```
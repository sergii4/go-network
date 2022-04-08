Run
```
$ go run main.go -total 10 -rps 1 -int 100
```
Check TCP connections
```
$ netstat -nc | grep :8080 | wc -l
```
Kill TCP connections
```
$ tcpkill -i eth0 port 8080
```






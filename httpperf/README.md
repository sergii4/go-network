# Intro
This folder contains http clients and servers as well as http tracer tool.

# Experiments
## HTTP1.1 and H2C

### Run HTTP1.1/H2C server
To make an experiment with http(1.1 and 2) run server:
```
$ cd h2csrv
$ go run main.go

```
This server allows you to [connect via HTTP1.1 then upgrade to HTTP/2 (H2C)](https://www.mailgun.com/blog/dev-life/http-2-cleartext-h2c-client-example-go/#chapter-1)
H2C is, it is essentially HTTP/2 but without TLS.

Both Golang HTTP client and server supports HTTP1.1 and 2, but for HTTP2 you need HTTPS according Golang [doc](https://cs.opensource.google/go/go/+/refs/tags/go1.18.1:src/net/http/transport.go;drc=3d7f83612390d913e7e8bb4ffa3dc69c41b3078d;l=74)

### Run low performance client 

To do first experiment you can run low performance client app. App which doesn't read and close response body
We run app that sends 10 request simultaneously, then wait 100 milliseconds, 100 request total
```
$ cd lowperfcli
$ go run main.go -total 100 -rps 10 -int 100
```

From server console:
```
http vesrion HTTP/1.1

```
From client console:
```
url http://localhost:8080/
total requests: 100
total reused connections: 0
reused connections / total request: 0.000000
```
we see zero that connection is reused. Because if you don't completely read the body of HTTP responses (because that will be "stuck" in the connection before the next responses' HTTP headers, and hence block everything behind it)

If we check how many connections are open and waiting:

```
$ netstat -nt | grep  8080 | grep -i time_wait  | wc -l 
```
Number should be
```
100
```

### Run HTTP client with default settings(read/close response body)
```
$ cd cli
$ go run main.go -total 1000 -rps 100 -int 100
```

From server console:
```
http vesrion HTTP/1.1

```
From client console:
```
url http://localhost:8080/
total requests: 1000
total reused connections: 320
reused connections / total request: 0.320000
```
We reused connection in 320 requests from 1000 and number of open and waiting connections could be about 944

### Run tuned HTTP client (set up max idle connections and max idle connections per host)
100 max idle connections and 100 max idle connections per host
```
$ go run main.go -total 1000 -rps 100 -int 100 -mic 100 -micph 100
```

From server console:
```
http vesrion HTTP/1.1

```
From client console:
```
url http://localhost:8080/
total requests: 1000
total reused connections: 903
reused connections / total request: 0.903000
```
We reused connection in 903 requests from 1000 and number of open and waiting connections should be about 100

For bigger number difference is even bigger

### Run HTTP client with HTTP/2 transport
```
$ go run main.go -total 1000 -rps 100 -int 100 -http2 true
```

From server console:
```
http vesrion HTTP/2.0

```
From client console:
```
url http://localhost:8080/
total requests: 1000
total reused connections: 999
reused connections / total request: 0.999000
```
We reused connection in 999(!) requests from 1000 and number of open and waiting connections should be just 1
```
$ netstat -nt | grep  8080 | grep -i time_wait  | wc -l
       1
```
## Run HTTP/2 over HTTPS
### Run HTTPS server 

```
$ cd sslsrv
$ go run maing.go

```
### Run HTTP client over HTTPS
```
$ go run main.go -total 1000 -rps 100 -int 100 -https true
```
From server console:
```
http vesrion HTTP/2.0

```
From client console:
```
url http://localhost:8080/
total requests: 1000
total reused connections: 999
reused connections / total request: 0.999000
```
We reused connection in 999(!) requests from 1000. It illustrates that Golang support HTTP/2 over HTTPS by defualt. 

# Summary

http client with defalt transport - t\
http client with http/2 transport - t2\
default http server - srv\
http server with upgrade to h2c - h2cSrv\
https server - httpsSrv

t  (http url)->    srv      ✅ http/1.1

t  (http url)->    h2cSrv   ✅ http1.1

t2 (http url)->    h2cSrv   ✅ http/2 (h2c)

t (https url)->    httpsSrv ✅ http/2

t2 (http url)->    srv      ❌

# wssocks

![build](https://github.com/genshen/wssocks/workflows/Go/badge.svg)

> socks5 over websocket.

wssocks can proxy TCP and UDP(not implemented currently) connections via socks5. But the socks5 data is wrapped in websockets and then sent to server.

## Features
- **Transfer data through firewalls**  
In some network environment, due to the restricts of firewalls, only http(s)/websocket is allowed. wssocks is mainly useful for passing through firewalls. We can access the inner netwrok (such as ssh) behind the firewalls via socks protocol wrapped in websockets.  
- **High performance**  
wssocks only create one TCP connection (websocket) per client to handle multiple socks5 connections, which achieves much higher performance.
- **Easy to use**  
No configures, no dependences, just a single executable including client and server.

## Build and install
```bash
cd status-web; yarn install; yarn build; cd ../
go get -u github.com/rakyll/statik
cd server; statik --src=../status-web/build/; cd ../
go build
go install
```
You can also download it from [release](https://github.com/genshen/wssocks/releases) page.

## Quick start

### server side
```bash
wssocks server --addr :1088
```
### client side
```bash
wssocks client --addr :1080 --remote ws://example.com:1088
# using ssh to connect to example.com which may be behind firewalls.
ssh -o ProxyCommand='nc -x 127.0.0.1:1080 %h %p' user@example.com 
```

And set your socks5 server address as `:1080` in your socks5 client (such as [proxifier](https://www.proxifier.com/) or proxy setting in mac's network preferences) if you need to use socks5 proxy in more situations, not only `ssh` in terminal.  

## Advanced usage
### enable http and https proxy
You can also enable http and https proxy by `--http` option(in client side)
if http(s) proxy in server side is enabled:

```bash
# client siede
wssocks client --addr :1080 --remote ws://example.com:1088 --http
```
The http proxy listen address is specified by `--http-addr` in client side (default value is `:1086`),
and https proxy listen address is the same as socks5 proxy listen address(specified by `--addr` option).

Then you can set server address of http and https proxy as `:1080` 
in your http(s) proxy client (e.g. mac's network preferences).

note: http(s) proxy is enabled by default in server side, you can disable it in server side 
by `wssocks server --addr :1088 --http=false` .

### Connection key
In some cases, you don't want anyone to connect to your wssocks server.
You can use connection key to prevent the clients who don't have correct connection authentication.  
At server side, just enable flag `--auth`, e.g.:
```bash
wssocks server --addr :1088 --auth
```
Then it will generate a random connection key.
You can also specific a customized connection key via flag `--auth_key`.  
At client side, connect to wssocks server via the connection key:
```bash
wssocks client --remote ws://example.com:1088 --key YOUR_CONNECTION_KEY
```

### TSL/SSL support
Method 1: 
In version 0.5.0, transfering data between wssocks client and wssocks server under TSL/SSL protocol is supported.

At server side, use `--tsl` flag to enable TSL/SSL support, 
and specific path of certificate via `--tls-cert-file` and `--tls-key-file`.
e.g.
```bash
wssocks server --addr :1088 --tsl --tls-cert-file /path/of/certificate-file --tls-key-file /path/of/certificate-key-file
```
At client side, we can then use `wss://example.com:1088` as remote address, for instance.

Method 2:
Use nginx reverse proxy, enable ssl and specific certificate file and certificate key file in nginx config.
For more information, see issue [#11](https://github.com/genshen/wssocks/issues/11#issuecomment-669324542)).

### Server status
In version 0.5.0, we can enable statue page of server by passing `--status` flag at server side (status page is disabled by default).  
Then, you can get server status in your browser of client side, by visiting http://example.com:1088/status (where example.com:1088 is the address of wssocks server).

### Help
```
wssocks --help
wssocks client --help
wssocks server --help
```

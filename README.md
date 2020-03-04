# wssocks
> socks5 over websocket.

wssocks can proxy TCP and UDP(not implemented currently) connections via socks5. But the socks5 data is wrapped in websockets and then sent to server.

## Features
- **Transporte data through firewalls**  
In some network environment, due to the restricts of firewalls, only http(s)/websocket is allowed. wssocks is mainly useful for passing through firewalls. We can access the inner netwrok (such as ssh) behind the firewalls via socks protocol wrapped in websockets.  
- **High performance**  
wssocks only create one TCP connection (websocket) per client to handle multiple socks5 connections, which achieves much higher performance.
- **Easy to use**  
No configures, no dependences, just a single executable including client and server.

## Install
```
go get -u github.com/genshen/wssocks
```
You can also download it from [release](https://github.com/genshen/wssocks/releases) page.

## Usage

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

### help
```
wssocks --help
wssocks client --help
wssocks server --help
```

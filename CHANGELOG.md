
<a name="v0.5.0-beta"></a>
## [v0.5.0-beta](https://github.com/genshen/wssocks/compare/v0.4.1...v0.5.0-beta)

> 2020-08-06

### Build

* **docker:** update Dockerfile to inject git revision, build data, go ver into output of 'version'
* **docker:** update Dockerfile to build static files of status-web and embed them into go binary
* **gomodule:** update package github.com/genshen/cmds to avoid to call exist(2) in subcommand help
* **gomodule:** update dependencies version
* **makefile:** make target `clean` and `all` as '.PHONY'

### Chore

* **status:** remove unused import in tsx files of 'status-web' for passing CI job

### Ci

* **action:** update github action config to build 'status-web' and generate build files to go file

### Docs

* **readme:** update building document in README.md
* **readme:** add github action badge to README.md file

### Feat

* use nhooyr.io/websocket as websocket lib to replace gorilla/websocket lib
* **client:** set timeout for sending heartbeat to wssocks server
* **client:** add skip-tls-verify to client to ignore tsl verification if server ssl/tls is enabled
* **client:** use more specified context in establish and http proxy client to send data to server
* **client:** use timeout context or cancel context to writing data to server
* **logs:** log only proxy connecting and closing message when a TTY is not attached on client side
* **server:** user can provide a user-specified connection key for authentication if auth is enabled
* **server:** add hub collection to store all hubs and websocket connections created at server side
* **server:** add --status flage to server cli to enable/disable status page
* **server:** let func DefaultProxyEst.establish return ConnCloseByClient err it is closed by client
* **status:** add static page for showing service status (not req data from server)
* **status:** fetch server info (not include statistics data) from server and show in info board
* **status:** show real server address and can copy address in status page
* **status:** fetch statistics data (e.g. uptime, clients) from server and shown in status page
* **status:** serve statis files of status page at server side
* **status:** fxi theme color and github libk stype of status page
* **status:** add logo files for status page
* **version:** add ability to show git commit hash in version sub-command
* **version:** write buildTime and go build version to version info when building via makefile
* **version:** update version to v0.5.0-beta

### Fix

* fix bug of `client close due to message too big` by increasing message read limit
* **logs:** dont show proxy connections dashboard/table when a TTY is not attached on client side
* **server:** add missing return statement in ServeHTTP after func websocker.Accept returns an error
* **server:** fix bug of infinite run loop after closing websocket/hub
* **server:** also perform tellClose if establish function return a non-ConnCloseByClient error
* **server:** fix "send on closed channel" error by removing tellClose with channel
* **server:** close proxy instances when closing hub (hub is closed when websocket is finished)
* **status:** git axios url of requesting status data of server, it should be '/api/status'

### Improvement

* **server:** generate random connection key in server side if connection key is enabled

### Merge

* Merge pull request [#18](https://github.com/genshen/wssocks/issues/18) from genshen/feature-more-specified-context
* Merge pull request [#15](https://github.com/genshen/wssocks/issues/15) from genshen/feature-web-status
* Merge pull request [#13](https://github.com/genshen/wssocks/issues/13) from genshen/fix-send-on-closed-channel

### Refactor

* rename file wss/client.go to wss/proxy_client_interface.go, move proxy client of https to new file
* **client:** call Client.Reply without onDial function as callback, move onDial out of Reply
* **client:** refactor func NewWebSocketClient: create WebSocketClient at the last line of func
* **client:** declare *net.TCPConn forward and pass to client.Reply in incoming socks5 handle
* **client:** use context to close heartbeat loop and client websocket income message listening
* **client:** move inlined onDial callback in Client.ListenAndServe in wssocks_client.go as func
* **logs:** refactor clear lines: move check IsTerminal to Writer from OS-platform's clearLines
* **server:** rm ctx passed to establish in ProxyEstablish, use timeout ctx in establish instead
* **server:** implement ServeHTTP interface for websocket serving to replace func handle
* **server:** implement BufferedWR using more elegant approach
* **server:** extract interface of establishing connections for socks5 and http proxy
* **server:** move proxy connections maintaining in server side to hub

### Pull Requests

* Merge pull request [#10](https://github.com/genshen/wssocks/issues/10) from genshen/websocket-lib-nhooyr.io/websocket
* Merge pull request [#8](https://github.com/genshen/wssocks/issues/8) from DefinitlyEvil/master

### BREAKING CHANGE


change first parameter in func BeforeRequest of ServerRedirect interface: from
gorilla websocket dialer to pointer of http.Client.

We also update the minimal go building version to 1.13 due to the usage of feature `errors.Is` to
handle results of command line parsing.
go version less then 1.12 (including 1.12) is not supported from this commit.


<a name="v0.4.1"></a>
## [v0.4.1](https://github.com/genshen/wssocks/compare/v0.4.0...v0.4.1)

> 2020-02-24

### Build

* **docker:** update version of golang docker image to 1.13.8
* **gomodule:** update go.sum file

### Fix

* **client:** log error, instead of fatal, when httpserver listener returns error
* **version:** fix core version not update, now update it to 0.4.1


<a name="v0.4.0"></a>
## [v0.4.0](https://github.com/genshen/wssocks/compare/v0.3.0...v0.4.0)

> 2020-02-11

### Build

* **gomodule:** update dependencies: go-isatty and crypto

### Feat

* **client:** handel kill signal in client closing
* **client:** stop all connections or tasks and exit if one of tasks is finished
* **client:** we can close client listener and heartbeat loop

### Refactor

* **logs:** split connection reecords updating and log writing

### Style

* **server:** code formating, change to use error wrapping


<a name="v0.3.0"></a>
## [v0.3.0](https://github.com/genshen/wssocks/compare/v0.3.0-alpha.2...v0.3.0)

> 2019-09-01

### Build

* **gomodule:** update dependenies.

### Docs

* update readme document and help messages.

### Feat

* **client:** add basic feature of http proxy by http protocol.
* **http:** add Hijacker http Proxy.
* **log:** better log to show http proxy size and welcome messages.
* **version:** add version negotiation plugin.
* **version:** send and check compatible version code in version negotiation.
* **version:** update version to 0.3.0 and protocol version to 0x003.

### Refactor

* rename ServerData.Type to ServerData.Tag and rename unused param conn in WebSocketClient.NewProxy.
* replace net.TCPConn with ReadWriteCloser in proxy connections container.
* **client:** use callback(not channel) to receive data from server.
* **client:** use channel to handle data received from proxy server.
* **server:** remove server channel, use callback instead.
* **server:** use channel to handle data received from proxy client.


<a name="v0.3.0-alpha.2"></a>
## [v0.3.0-alpha.2](https://github.com/genshen/wssocks/compare/v0.3.0-alpha...v0.3.0-alpha.2)

> 2019-08-27

### Docs

* **readme:** update README, add document of http proxy.

### Feat

* **logs:** better logs for version negotiation.
* **version:** update version to 0.3.0-alpha.2

### Refactor

* **logs:** better server logs of connections size.


<a name="v0.3.0-alpha"></a>
## [v0.3.0-alpha](https://github.com/genshen/wssocks/compare/v0.2.1...v0.3.0-alpha)

> 2019-08-26

### Build

* add github workflows for building go.

### Feat

* **http:** add http proxy support.
* **http:** add https proxy support.
* **version:** update version to 0.3.0-alpha.

### Merge

* Merge pull request [#7](https://github.com/genshen/wssocks/issues/7) from genshen/feature-http-proxy
* **ticker:** Merge pull request [#6](https://github.com/genshen/wssocks/issues/6) from genshen/remove-ticker

### Perf

* **ticker:** remove ticker support in both client and server.

### Refactor

* **client:** reorganize data types and functions in client implementation.
* **client:** add interface for different proxy(socks5, http, https) in client.


<a name="v0.2.1"></a>
## [v0.2.1](https://github.com/genshen/wssocks/compare/v0.2.0...v0.2.1)

> 2019-06-16

### Build

* **makefile:** add linux arm64 building in makefile.

### Feat

* **docker:** add Dockerfile.
* **logs:** better view for proxy connections dashboard.
* **logs:** adapte client normal running logs to progress logs.
* **logs:** add feature of progress logs of proxy connections.
* **logs:** add logrus lib as log lib.
* **version:** update version to 0.2.1

### Fix

* **windows:** fix windows building problems.

### Merge

* **logs:** Merge pull request [#5](https://github.com/genshen/wssocks/issues/5) from genshen/dev

### Refactor

* **log:** use ssh/terminal pacakge to get terminal size.


<a name="v0.2.0"></a>
## [v0.2.0](https://github.com/genshen/wssocks/compare/v0.1.0...v0.2.0)

> 2019-04-11

### Build

* **makefile:** add PACKAGE option to makefile and go build command.

### Docs

* **license:** add MIT license.

### Feat

* **plugin:** add *websocket.Dialer as func param in client plugin interface ServerRedirect.
* **version:** change version name to v0.2.0

### Fix

* **version:** client also sends its version information to server now.

### Refactor

* **client:** refactor client code: move socks listerning code to file wss/wssocks.go

### Pull Requests

* Merge pull request [#2](https://github.com/genshen/wssocks/issues/2) from genshen/dev


<a name="v0.1.0"></a>
## [v0.1.0](https://github.com/genshen/wssocks/compare/v0.1.0-vpn...v0.1.0)

> 2019-03-03

### Chore

* **socks5:** add more legality checking for socks5 server side.

### Docs

* **readme:** add README

### Feat

* **heartbeat:** add feature of websocket heart beat.
* **log:** add log for parsing error of cli.
* **plugin:** add client Plugin interface.
* **protocol:** add check of protocol version incompatibility.

### Fix

* **close:** add more connection close calling.
* **server:** fix connection lose error(read connection EOF)

### Merge

* Merge pull request [#1](https://github.com/genshen/wssocks/issues/1) from genshen/dev

### Refactor

* **datatypes:** rename WriteProxyMessage func in ConcurrentWebSocket and move type Base64WSBuff
* **datatypes:** combine data typeRequestMessage anf ProxyData into one type.
* **server:** return error in dispatchMessage not nil and move mutex forward in Flush.
* **server:** remove unnercessary terms in ServerWS.map[]Connector
* **server:** add function NewServerWS and rename NewConn to AddConn.

### Style

* reorganize the code structural.
* move code position like println and comments.
* move directory from ws-socks to wssocks.


<a name="v0.1.0-vpn"></a>
## v0.1.0-vpn

> 2019-02-08

### Feat

* **cmd:** add user command line option to set ticker.
* **server:** add ticker and non-ticker option implementation for proxy server.

### Fix

* fixed known bugs

### Perf

* **websocket:** use one websocket for all connections, instead of one websocket for one socks.

### Refactor

* rename and remove variables
* **client:** rename client command variable, and rename ws-socks/client.go -> ws-socks/socks5_server.go



<a name="v0.6.0"></a>
## [v0.6.0](https://github.com/genshen/wssocks/compare/v0.5.0...v0.6.0)

> 2023-07-31

### Build

* **gomodule:** bump go packages to latest
* **gomodule:** bump package golang.org/x/crypto and golang.org/x/sync to fix building error
* **gomodule:** update dependencies version
* **makefile:** add darwin-arm64 building in Makefile
* **npm:** bump react-scripts to v5 to fix "error:0308010C:digital envelope routines::unsupported"
* **status:** bump dependencies version in yarn.lock
* **status:** bump evergreen-ui to 6.5.1

### Ci

* **github-action:** bump gh-actions version: go, node, actions of setup-go, setup-node and checkout
* **github-action:** create Release via github action
* **github-action:** bump node to 16.x and go to 1.18 in github action

### Docs

* **changelog:** add changelog for version 0.6.0

### Feat

* **client:** remove the unused parameter `*sync.Once` in `Handles.Wait` of client
* **client:** add ability to catch error from any tasks in main goroutine
* **version:** bump version to v0.6.0

### Fix

* **client:** fix the incorrect error passing in client side data transmission
* **server:** fix bug of server side crashing when using http/https proxy

### Merge

* **client:** Merge pull request [#69](https://github.com/genshen/wssocks/issues/69) from genshen/remove-client-Wait-parameter
* **client:** Merge pull request [#68](https://github.com/genshen/wssocks/issues/68) from genshen/fix-error-passing-in-client
* **client:** Merge pull request [#54](https://github.com/genshen/wssocks/issues/54) from genshen/feature-client-wait-with-error
* **github-action:** Merge pull request [#67](https://github.com/genshen/wssocks/issues/67) from genshen/ci-create-release
* **server:** Merge pull request [#57](https://github.com/genshen/wssocks/issues/57) from genshen/fix-http-proxy-panic
* **status:** Merge pull request [#45](https://github.com/genshen/wssocks/issues/45) from genshen/bump-status-web-dependencies


<a name="v0.5.0"></a>
## [v0.5.0](https://github.com/genshen/wssocks/compare/v0.5.0-rc.3...v0.5.0)

> 2021-01-25

### Build

* **docker:** upgrade go/node/alpine versions in Dockerfile for docker building
* **npm:** bump react dependency of status web page to v17 via cra, and enable pwa

### Ci

* **gh-action:** upgrade go and node versions in github action building

### Docs

* **changelog:** add changelog for version 0.5.0

### Feat

* **version:** bump version to v0.5.0


<a name="v0.5.0-rc.3"></a>
## [v0.5.0-rc.3](https://github.com/genshen/wssocks/compare/v0.5.0-rc.2...v0.5.0-rc.3)

> 2021-01-08

### Chore

* **logs:** add description to func ProgressLog.SetLogBuffer

### Docs

* **changelog:** add changelog for version 0.5.0-rc.3

### Feat

* **client:** better error when there is error in Client.ListenAndServe (in wss/wssocks_client.go)
* **plugin:** add "connection option" plugin interface
* **version:** bump version to v0.5.0-rc.3

### Refactor

* **plugin:** rename struct type "Plugin" to "Plugins"

### Revert

* **version:** set protocol version back to 0x004, because it has been set in v0.5.0-beta


<a name="v0.5.0-rc.2"></a>
## [v0.5.0-rc.2](https://github.com/genshen/wssocks/compare/v0.5.0-rc.1...v0.5.0-rc.2)

> 2021-01-02

### Build

* **docker:** fix docker build error of "../web-build: no such file or directory"

### Docs

* **changelog:** add changelog for version 0.5.0-rc.2

### Feat

* **plugin:** return error, instead of calling log.Fatal, when the adding plugin exists
* **version:** bump version to v0.5.0-rc.2 and increase version code

### Fix

* **server:** fix compiling error of no package "github.com/genshen/wssocks/cmd/server/statik"

### Refactor

* **plugin:** rename plugin api: AddPluginRedirect -> AddPluginRequest


<a name="v0.5.0-rc.1"></a>
## [v0.5.0-rc.1](https://github.com/genshen/wssocks/compare/v0.5.0-beta.3...v0.5.0-rc.1)

> 2021-01-02

### Build

* **npm:** upgrade npm dependencies
* **status:** update dependencies (axios and evergreen-ui) for status page

### Ci

* **action:** fix building error in github action while performing static files to go code generation

### Docs

* **changelog:** add changelog for version 0.5.0-rc.1

### Feat

* **client:** use http.DefaultTransport based Transport in http client for http dialing
* **plugin:** we can change value of http transport (used for websocket dialing) in request plugin
* **version:** bump version to v0.5.0-rc.1

### Fix

* **client:** fix unexpected closing of client by using lock to Write and context canceling
* **status:** fix building error of "data Object is possibly 'undefined'."

### Refactor

* **cli:** move/split cli implementation of client and server to cmd directory
* **client:** use more semantic variable names in client Options
* **client:** move client connections closing to func `NotifyClose` in struct Handles
* **client:** move sync.WaiitGroup passed to StartClient and Wait as a field of type Handles
* **client:** split client setting-up func StartClient into multiple function calls
* **plugin:** rename plugin ServerRedirect to RequestPlugin


<a name="v0.5.0-beta.3"></a>
## [v0.5.0-beta.3](https://github.com/genshen/wssocks/compare/v0.5.0-beta.2...v0.5.0-beta.3)

> 2020-10-03

### Build

* **docker:** update go version in docker building, and specific alpine version
* **gomodule:** update dependencies version

### Chore

* add go code report badge

### Docs

* **changelog:** add changelog for version 0.5.0-beta.3

### Feat

* **version:** bump version to v0.5.0-beta.3

### Fix

* **server:** increase server read limit to 8 MiB to fix client exit with error "StatusMessageTooBig"

### Style

* format project code: use tab as indent


<a name="v0.5.0-beta.2"></a>
## [v0.5.0-beta.2](https://github.com/genshen/wssocks/compare/v0.5.0-beta...v0.5.0-beta.2)

> 2020-08-23

### Build

* **status:** update dependencies in status page: evergreen-ui to 5.x and typescript to 3.9

### Chore

* fix typo of UI (one for status page and one for connections table of client)

### Docs

* **changelog:** add changelog for version 0.5.0-beta.2
* **changelog:** add changelog generated by git-chglog tool
* **readme:** add badges of docker image size, version and pulls
* **readme:** add document of "connection key", "TSL/SSL support" and "server status"

### Feat

* **client:** add flag for passing user defined http headers to websocket request and send to remote
* **server:** add flags to server sub-command to support HTTPS/tls: -tls -tls-cert-file -tls-key-file
* **server:** add ability of setting websocket serving path in server cli
* **status:** show correct server address (including proctocol and base path) in status page
* **version:** update version to v0.5.0-beta.2

### Fix

* **server:** remove channel usage in server side to avoid panic "send on closed channel"

### Merge

* Merge pull request [#25](https://github.com/genshen/wssocks/issues/25) from genshen/feature-ssl-tsl-support

### Pull Requests

* Merge pull request [#27](https://github.com/genshen/wssocks/issues/27) from genshen/fix-server-crashed-if-client-killed


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


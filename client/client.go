package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/genshen/wssocks/wss"
	"github.com/genshen/wssocks/wss/term_view"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

func NewHttpClient() (*http.Client, *http.Transport) {
	// set to use default Http Transport
	tr := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	httpClient := http.Client{
		Transport: &tr,
	}
	return &httpClient, &tr
}

type Options struct {
	LocalSocks5Addr string      // local listening address
	RemoteUrl       *url.URL    // url of server
	RemoteHeaders   http.Header // parsed websocket headers (not presented in flag).
	ConnectionKey   string      // connection key for authentication
	SkipTLSVerify   bool        // skip TSL verify
}

type Handles struct {
	wsc    []*wss.WebSocketClient
	hb     *wss.HeartBeat
	cl     *wss.Client
	closed bool
	wg     *sync.WaitGroup
}

func NewClientHandles() *Handles {
	return &Handles{closed: true, wg: &sync.WaitGroup{}}
}

// send closing message to all running tasks
func (hdl *Handles) NotifyClose(once *sync.Once, wait bool) {
	if hdl.closed {
		return
	}
	hdl.closed = true

	// stop tasks in signal
	once.Do(func() {
		if hdl.cl != nil {
			hdl.cl.Close(wait)
		}
		hdl.hb.Close()
		for _, wsc := range hdl.wsc {
			wsc.Close()
		}
	})
}

// create a server websocket connection based on user options.
func (hdl *Handles) CreateServerConn(c *Options, ctx context.Context) ([]*wss.WebSocketClient, error) {
	if c.ConnectionKey != "" {
		c.RemoteHeaders.Set("Key", c.ConnectionKey)
	}

	httpClient, transport := NewHttpClient()

	if c.RemoteUrl.Scheme == "wss" && c.SkipTLSVerify {
		// ignore insecure verify
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		log.Warnln("Warning: you have skipped verification of the server's certificate chain and host name. " +
			"Then client will accepts any certificate presented by the server and any host name in that certificate. " +
			"In this mode, TLS is susceptible to man-in-the-middle attacks.")
	}

	// load and use option plugin
	if clientPlugin.HasOptionPlugin() {
		if err := clientPlugin.OptionPlugin.OnOptionSet(*c); err != nil {
			return nil, err
		}
	}

	// loading and execute plugin
	if clientPlugin.HasRequestPlugin() {
		// in the plugin, we may add http header/dialer and modify remote address.
		if err := clientPlugin.RequestPlugin.BeforeRequest(httpClient, transport, c.RemoteUrl, &c.RemoteHeaders); err != nil {
			return nil, err
		}
	}

	var wsc []*wss.WebSocketClient
	// start websocket connection (to remote server).
	for i := 0; i < 4; i++ {
		w, err := wss.NewWebSocketClient(ctx, c.RemoteUrl.String(), httpClient, c.RemoteHeaders)
		if err != nil {
			return nil, fmt.Errorf("establishing connection error: %w", err)
		}
		wsc = append(wsc, w)
	}

	// todo chan for wsc and tcp accept
	hdl.wsc = wsc
	return wsc, nil
}

func (hdl *Handles) NegotiateVersion(ctx context.Context, remoteUrl string) error {
	// negotiate version
	if version, err := wss.ExchangeVersion(ctx, hdl.wsc[0].WsConn); err != nil {
		return err
	} else {
		for i, w := range hdl.wsc {
			if i == 0 {
				continue
			}
			wss.ExchangeVersion(ctx, w.WsConn) //一定要交互一次
		}
		if clientPlugin.HasVersionPlugin() {
			if err := clientPlugin.VersionPlugin.OnServerVersion(version); err != nil {
				return err
			}
		} else {
			log.WithFields(log.Fields{
				"compatible version code": version.CompVersion,
				"version code":            version.VersionCode,
				"version number":          version.Version,
			}).Info("server version")

			// client protocol version must eq or smaller than server version (newer client is not allowed)
			// And, compatible version is the lowest version for client.
			if version.CompVersion > wss.VersionCode || wss.VersionCode > version.VersionCode {
				return errors.New("incompatible protocol version of client and server")
			}
			if version.Version != wss.CoreVersion {
				log.WithFields(log.Fields{
					"client wssocks version": wss.CoreVersion,
					"server wssocks version": version.Version,
				}).Warning("different version of client and server wssocks")
			}
			if version.EnableStatusPage {
				if endpoint, err := url.Parse(remoteUrl + "/status"); err != nil {
					return err
				} else {
					endpoint.Scheme = "http"
					log.WithFields(log.Fields{
						"endpoint": endpoint.String(),
					}).Infoln("server status is available, you can visit the endpoint to get status.")
				}
			}
		}
	}
	return nil
}

func (hdl *Handles) StartClient(c *Options, once *sync.Once) {
	// wait group wait for one of go func
	hdl.wg.Add(2) // wait for all go func

	// stop all connections or tasks, if one of tasks is finished.
	closeAll := func() {
		if hdl.cl != nil {
			hdl.cl.Close(false)
		}
		hdl.hb.Close()
		for _, wsc := range hdl.wsc {
			wsc.Close()
		}
	}

	// 接收服务器下发消息
	for _, w := range hdl.wsc {
		hdl.wg.Add(1)
		go func() {
			defer hdl.wg.Done()
			defer once.Do(closeAll)
			if err := w.ListenIncomeMsg(1 << 29); err != nil {
				log.Error("error websocket read:", err)
			}
		}()

	}

	heartbeat, hbCtx := wss.NewHeartBeat(hdl.wsc)
	hdl.hb = heartbeat
	// send heart beats.
	go func() {
		defer hdl.wg.Done()
		defer once.Do(closeAll)
		if err := hdl.hb.Start(hbCtx, time.Minute); err != nil {
			log.Info("heartbeat ending", err)
		}
	}()

	record := wss.NewConnRecord()
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		// if it is tty, use term_view as output, and set onChange function to update output
		plog := term_view.NewPLog(record)
		log.SetOutput(plog) // change log stdout to plog

		record.OnChange = func(wss.ConnStatus) {
			// update log
			plog.SetLogBuffer(record) // call Writer.Write() to set log data into buffer
			plog.Writer.Flush(nil)    // flush buffer
		}
	} else {
		record.OnChange = func(status wss.ConnStatus) {
			if status.IsNew {
				log.WithField("address", status.Address).Traceln("new proxy connection")
			} else {
				log.WithField("address", status.Address).Traceln("close proxy connection")
			}
		}
	}

	// start listen for socks5 and https connection.
	hdl.cl = wss.NewClient()
	go func() {
		defer hdl.wg.Done()
		defer once.Do(closeAll)
		if err := hdl.cl.ListenAndServe(record, hdl.wsc, c.LocalSocks5Addr, func() {
			log.WithField("socks5 listen address", c.LocalSocks5Addr).
				Info("listening on local address for incoming proxy requests.")
		}); err != nil {
			log.Errorln(err)
		}
	}()
	hdl.closed = false
}

func (hdl *Handles) Wait(once *sync.Once) {
	go func() {
		firstInterrupt := true
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		for { // accept multiple signal
			select {
			case <-c:
				if firstInterrupt {
					log.Println("press CTRL+C to force exit")
					firstInterrupt = false
					hdl.NotifyClose(once, true)
				} else {
					os.Exit(0)
				}
			}
		}
	}()

	hdl.wg.Wait() // wait all tasks finished
	// about exit: 1. press ctrl+c, it will wait active connection to finish.
	// 2. press twice, force exit.
	// 3. one of tasks error, exit immediately.
	// 4. close server, then client exit (the same as 3).
}

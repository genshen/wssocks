package status

import (
    "encoding/json"
    "github.com/genshen/wssocks/wss"
    "net/http"
    "time"
)

type Version struct {
    VersionStr  string `json:"version_str"`
    VersionCode int    `json:"version_code"`
    ComVersion  int    `json:"compatible_version"`
}

type Info struct {
    Version              Version `json:"version"`
    Socks5Enable         bool    `json:"socks5_enabled"`
    Socks5DisableReason  string  `json:"socks5_disabled_reason"`
    HttpsEnable          bool    `json:"http_enabled"`
    HttpsDisableReason   string  `json:"http_disabled_reason"`
    SSLEnable            bool    `json:"ssl_enabled"`
    SSLDisableReason     string  `json:"ssl_disabled_reason"`
    ConnKeyEnable        bool    `json:"conn_key_enable"`
    ConnKeyDisableReason string  `json:"conn_key_disabled_reason"`
}

type Statistics struct {
    UpTime  float64 `json:"up_time"`
    Clients int     `json:"clients"`
    Proxies int     `json:"proxies"`
}

type Status struct {
    Info       Info       `json:"info"`
    Statistics Statistics `json:"statistics"`
}

type handleStatus struct {
    enableHttp    bool
    enableConnKey bool
    hc            *wss.HubCollection
    setupTime     time.Time
}

// create a http handle for handling service status
func NewStatusHandle(hc *wss.HubCollection, enableHttp bool, enableConnKey bool) *handleStatus {
    return &handleStatus{
        hc:            hc,
        enableHttp:    enableHttp,
        enableConnKey: enableConnKey,
        setupTime:     time.Now(),
    }
}

func (s *handleStatus) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*") // todo: remove in production env
    w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
    w.Header().Set("Content-Type", "application/json")

    clients, proxies := s.hc.GetConnCount()
    duration := time.Now().Sub(s.setupTime).Truncate(time.Second)

    status := Status{
        Info: Info{
            Version: Version{
                VersionStr:  wss.CoreVersion,
                VersionCode: wss.VersionCode,
                ComVersion:  wss.CompVersion,
            },
            Socks5Enable:        true,
            Socks5DisableReason: "",
            HttpsEnable:         s.enableHttp,
            ConnKeyEnable:       s.enableConnKey,
            SSLEnable:           false,
            SSLDisableReason:    "not support", // todo ssl support
        },
        Statistics: Statistics{
            UpTime:  duration.Seconds(),
            Clients: clients,
            Proxies: proxies,
        },
    }

    if !status.Info.HttpsEnable {
        status.Info.HttpsDisableReason = "disabled"
    }
    if !status.Info.ConnKeyEnable {
        status.Info.ConnKeyDisableReason = "disabled"
    }

    if err := json.NewEncoder(w).Encode(status); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
    }
}

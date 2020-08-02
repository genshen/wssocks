package status

import (
    "encoding/json"
    "github.com/genshen/wssocks/wss"
    "net/http"
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
    UpDays int `json:"up_days"`
}

type Status struct {
    Info       Info       `json:"info"`
    Statistics Statistics `json:"statistics"`
}

type handleStatus struct {
    enableHttp    bool
    enableConnKey bool
}

func NewStatusHandle(enableHttp bool, enableConnKey bool) *handleStatus {
    return &handleStatus{enableHttp: enableHttp, enableConnKey: enableConnKey}
}

func (s *handleStatus) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*") // todo: remove in production env
    w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
    w.Header().Set("Content-Type", "application/json")
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
            UpDays: 1,
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

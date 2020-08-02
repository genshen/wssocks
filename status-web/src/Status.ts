export interface Version {
    version_str: string
    version_code: number
    compatible_version: number
}

export interface Info {
    version: Version
    socks5_enabled: boolean
    socks5_disabled_reason: string
    http_enabled: boolean
    http_disabled_reason: string
    ssl_enabled: boolean
    ssl_disabled_reason: string
    conn_key_enable: boolean
    conn_key_disabled_reason: string
}

export interface Statistics {
    up_time: number
    clients: number
    proxies: number
}

export interface WssosksStatus {
    info: Info
    statistics: Statistics
}

package tg_client

import (
	"golang.org/x/net/proxy"
	"strconv"
)

type Proxy struct {
	Ip       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (p *Proxy) GetDial() proxy.ContextDialer {
	var auth *proxy.Auth
	if p.Username != "" && p.Password != "" {
		auth = &proxy.Auth{
			User:     p.Username,
			Password: p.Password,
		}
	}
	socks5, _ := proxy.SOCKS5("tcp", p.Ip+":"+strconv.Itoa(p.Port), auth, proxy.Direct)
	return socks5.(proxy.ContextDialer)
}

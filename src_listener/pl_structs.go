package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type HTTPConfig struct {
	HostBind string `json:"host_bind"`
	PortBind int    `json:"port_bind"`
	Uri      string `json:"uri"`
	Ssl      bool   `json:"ssl"`

	SslCert     []byte `json:"ssl_cert"`
	SslKey      []byte `json:"ssl_key"`
	SslCertPath string
	SslKeyPath  string

	Addresses string `json:"addresses"`
}

type HTTP struct {
	GinEngine *gin.Engine
	Server    *http.Server
	Config    HTTPConfig
	Name      string
	Active    bool
	mu        sync.Mutex
}

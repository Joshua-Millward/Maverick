package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"

	ax "github.com/Adaptix-Framework/axc2"
)

func HandlerCreateListenerDataAndStart(name string, data string, listenerCustomData []byte) (ax.ExtenderListener, ax.ListenerData, []byte, *Listener, error) {
	var cfg HTTPConfig
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, ax.ListenerData{}, nil, nil, fmt.Errorf("failed to parse config: %v", err)
	}

	if cfg.Uri == "" {
		cfg.Uri = "/"
	}
	if !strings.HasPrefix(cfg.Uri, "/") {
		cfg.Uri = "/" + cfg.Uri
	}

	protocol := "http"
	if cfg.Ssl {
		protocol = "https"
	}

	serverIP := cfg.HostBind
	if serverIP == "" || serverIP == "0.0.0.0" {
		serverIP = getOutboundIP()
	}
	cfg.Addresses = fmt.Sprintf("%s:%d", serverIP, cfg.PortBind)

	transport := &HTTP{
		Config: cfg,
		Name:   name,
	}

	listener := &Listener{transport: transport}

	listenerData := ax.ListenerData{
		Protocol:  protocol,
		BindHost:  cfg.HostBind,
		BindPort:  fmt.Sprintf("%d", cfg.PortBind),
		AgentAddr: cfg.Addresses,
		Status:    "Listen",
	}

	customData, _ := json.Marshal(cfg)

	return listener, listenerData, customData, listener, nil
}

func HandlerEditListenerData(name string, value any, config string) (ax.ListenerData, []byte, bool) {
	listener, ok := value.(*Listener)
	if !ok || listener.transport.Name != name {
		return ax.ListenerData{}, nil, false
	}

	var cfg HTTPConfig
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return ax.ListenerData{}, nil, false
	}

	protocol := "http"
	if cfg.Ssl {
		protocol = "https"
	}

	listenerData := ax.ListenerData{
		Protocol:  protocol,
		BindHost:  cfg.HostBind,
		BindPort:  fmt.Sprintf("%d", cfg.PortBind),
		AgentAddr: cfg.Addresses,
		Status:    "Listen",
	}

	customData, _ := json.Marshal(cfg)
	return listenerData, customData, true
}

func HandlerListenerStop(name string, value any) (bool, error) {
	listener, ok := value.(*Listener)
	if !ok || listener.transport.Name != name {
		return false, nil
	}

	err := listener.transport.Stop()
	return true, err
}

func (handler *HTTP) Start(ts Teamserver) error {
	if handler.Active {
		return nil
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.POST(handler.Config.Uri, handler.process_request)

	handler.Server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", handler.Config.HostBind, handler.Config.PortBind),
		Handler: router,
	}

	errChan := make(chan error, 1)

	if handler.Config.Ssl {
		fmt.Printf("   Started listener '%s': https://%s:%d%s\n", handler.Name, handler.Config.HostBind, handler.Config.PortBind, handler.Config.Uri)

		listenerPath := ListenerDataDir + "/" + handler.Name
		if _, err := os.Stat(listenerPath); os.IsNotExist(err) {
			os.Mkdir(listenerPath, os.ModePerm)
		}

		handler.Config.SslCertPath = listenerPath + "/listener.crt"
		handler.Config.SslKeyPath = listenerPath + "/listener.key"

		if len(handler.Config.SslCert) == 0 || len(handler.Config.SslKey) == 0 {
			if err := handler.gen_self_signed_cert(handler.Config.SslCertPath, handler.Config.SslKeyPath); err != nil {
				handler.Active = false
				return err
			}
		} else {
			os.WriteFile(handler.Config.SslCertPath, handler.Config.SslCert, 0600)
			os.WriteFile(handler.Config.SslKeyPath, handler.Config.SslKey, 0600)
		}

		cert, err := tls.LoadX509KeyPair(handler.Config.SslCertPath, handler.Config.SslKeyPath)
		if err != nil {
			handler.Active = false
			return fmt.Errorf("failed to load certificate: %v", err)
		}

		handler.Server.TLSConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			MaxVersion:   tls.VersionTLS13,
			Certificates: []tls.Certificate{cert},
		}

		go func() {
			err := handler.Server.ListenAndServeTLS("", "")
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- err
				return
			}
			errChan <- nil
		}()
	} else {
		fmt.Printf("   Started listener '%s': http://%s:%d%s\n", handler.Name, handler.Config.HostBind, handler.Config.PortBind, handler.Config.Uri)

		go func() {
			err := handler.Server.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- err
				return
			}
			errChan <- nil
		}()
	}

	select {
	case err := <-errChan:
		if err != nil {
			handler.Active = false
			return fmt.Errorf("failed to start listener: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		handler.Active = true
	}

	return nil
}

func (handler *HTTP) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	listenerPath := ListenerDataDir + "/" + handler.Name
	if _, err := os.Stat(listenerPath); err == nil {
		os.RemoveAll(listenerPath)
	}

	return handler.Server.Shutdown(ctx)
}

// process_request handles incoming agent POST requests
//
// Wire format (receive):
//   [36B agent_id/uuid][RC4(payload)][16B session_key (first checkin only)]
//
// Wire format (send):
//   [36B agent_id][RC4(response)]
func (handler *HTTP) process_request(ctx *gin.Context) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil || len(body) < 36 {
		ctx.Status(http.StatusBadRequest)
		return
	}

	agentIdBytes := body[:36]
	agentIdStr := string(agentIdBytes[:8])

	agentExist := ModuleObject.ts.TsAgentIsExists(agentIdStr)

	if !agentExist {
		// New agent — key is last 16 bytes
		if len(body) < 52 {
			ctx.Status(http.StatusBadRequest)
			return
		}

		key := make([]byte, 16)
		copy(key, body[len(body)-16:])
		encryptedData := body[36 : len(body)-16]

		// Save key
		err := ModuleObject.ts.TsExtenderDataSave(handler.Name, "key_"+agentIdStr, key)
		if err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}

		// Decrypt checkin
		crypt := NewMvCrypt(key)
		payload := crypt.Decrypt(encryptedData)

		fmt.Printf("[NEW AGENT] %s - Key: %02x\n", agentIdStr, key)

		// Create agent
		agentDataRes, err := ModuleObject.ts.TsAgentCreate(AGENT_WATERMARK, agentIdStr, payload, handler.Name, ctx.Request.Host, true)
		if err != nil {
			fmt.Printf("[ERROR] Failed to create agent: %v\n", err)
			ctx.Status(http.StatusInternalServerError)
			return
		}

		newAgentID := agentDataRes.Id

		// Save key for new ID if different
		if newAgentID != agentIdStr {
			ModuleObject.ts.TsExtenderDataSave(handler.Name, "key_"+newAgentID, key)
		}

		// Build response: [36B padded agent_id][RC4(newID)]
		randomID := make([]byte, 19)
		rand.Read(randomID)
		newID := []byte(agentDataRes.Id + hex.EncodeToString(randomID))

		encrypted := crypt.Encrypt(newID)
		response := append(agentIdBytes, encrypted...)

		ctx.Data(http.StatusOK, "application/octet-stream", response)
		return
	}

	// Existing agent
	_ = ModuleObject.ts.TsAgentSetTick(agentIdStr, handler.Name)

	agentKey, err := ModuleObject.ts.TsExtenderDataLoad(handler.Name, "key_"+agentIdStr)
	if err != nil || len(agentKey) != 16 {
		ctx.Status(http.StatusBadRequest)
		return
	}

	encryptedData := body[36:]
	crypt := NewMvCrypt(agentKey)
	payload := crypt.Decrypt(encryptedData)

	if len(payload) == 0 {
		ctx.Status(http.StatusOK)
		return
	}

	taskAction := payload[0]

	switch taskAction {
	case TASK_GET:
		hostedData, err := ModuleObject.ts.TsAgentGetHostedAll(agentIdStr, 0x12c0000)
		if err != nil || len(hostedData) == 0 {
			ctx.Status(http.StatusOK)
			return
		}

		encrypted := crypt.Encrypt(hostedData)
		response := append(agentIdBytes, encrypted...)
		ctx.Data(http.StatusOK, "application/octet-stream", response)

	case TASK_RESULT:
		if len(payload) > 1 {
			ModuleObject.ts.TsAgentProcessData(agentIdStr, payload[1:])
		}
		ctx.Status(http.StatusOK)

	default:
		ctx.Status(http.StatusOK)
	}
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		addrs, _ := net.InterfaceAddrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func (handler *HTTP) gen_self_signed_cert(certFile, keyFile string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{handler.Config.HostBind},
	}

	certData, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	var certBuffer bytes.Buffer
	pem.Encode(&certBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	handler.Config.SslCert = certBuffer.Bytes()
	os.WriteFile(certFile, handler.Config.SslCert, 0644)

	var keyBuffer bytes.Buffer
	pem.Encode(&keyBuffer, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	handler.Config.SslKey = keyBuffer.Bytes()
	os.WriteFile(keyFile, handler.Config.SslKey, 0644)

	return nil
}

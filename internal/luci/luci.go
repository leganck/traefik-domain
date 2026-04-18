package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	rpcPath  = "/cgi-bin/luci/rpc/"
	authPath = rpcPath + "auth"
	uciPath  = rpcPath + "uci"

	methodLogin = "login"
)

var (
	ErrRpcLoginFail        = errors.New("rpc: login fail")
	ErrHttpUnauthenticated = errors.New("http: Unauthenticated")
	ErrHttpUnauthorized    = errors.New("http: Unauthorized")
	ErrHttpForbidden       = errors.New("http: Forbidden")
)

type LuciClient struct {
	config     *OpenWRTConfig
	token      string
	httpClient *http.Client
	rpcID      int
}

type OpenWRTConfig struct {
	Hostname           string
	Username           string
	Password           string
	Port               int
	UseSSL             bool
	InsecureSkipVerify bool
	Timeout            int
}

type luciPayload struct {
	ID     int      `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type luciResponse struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

type DnsRecord struct {
	Type   string `json:".type"`
	Name   string `json:"name"`
	IP     string `json:"ip"`
	CName  string `json:"cname"`
	Target string `json:"target"`
	Remark string `json:"remark"`
}

const RecordRemark = "traefik-domain"

func NewLuciClient(rawURL, username, password string) (*LuciClient, error) {
	cfg, err := parseOpenWRTURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse openwrt host failed: %v", err)
	}
	cfg.Username = username
	cfg.Password = password

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.InsecureSkipVerify,
			},
			Dial: (&net.Dialer{
				Timeout:   time.Duration(cfg.Timeout) * time.Second,
				KeepAlive: time.Duration(cfg.Timeout) * time.Second,
			}).Dial,
		},
	}

	return &LuciClient{
		config:     cfg,
		httpClient: httpClient,
		rpcID:      1,
	}, nil
}

func parseOpenWRTURL(rawURL string) (*OpenWRTConfig, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}

	return &OpenWRTConfig{
		Hostname: host,
		Port:     portInt,
		UseSSL:   u.Scheme == "https",
		Timeout:  10,
	}, nil
}

func (c *LuciClient) Auth(ctx context.Context) error {
	resp, err := c.rpc(ctx, authPath, methodLogin, []string{c.config.Username, c.config.Password})
	if err != nil {
		return err
	}

	if resp == "null" {
		return ErrRpcLoginFail
	}

	c.token = resp
	return nil
}

func (c *LuciClient) rpc(ctx context.Context, path, method string, params []string) (string, error) {
	data, err := json.Marshal(luciPayload{
		ID:     c.rpcID,
		Method: method,
		Params: params,
	})
	if err != nil {
		return "", err
	}

	url := c.getURL(path, method)
	respBody, err := c.call(ctx, url, data)
	if err != nil {
		return "", err
	}

	var response luciResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", err
	}

	if response.Error != nil {
		return "", parseError(response.Error)
	}

	if response.Result != nil {
		return parseString(response.Result)
	}

	return "", nil
}

func (c *LuciClient) getURL(path, method string) string {
	proto := "https://"
	if !c.config.UseSSL {
		proto = "http://"
	}

	url := proto + c.config.Hostname + ":" + strconv.Itoa(c.config.Port) + path
	if method != methodLogin && c.token != "" {
		url = url + "?auth=" + c.token
	}

	return url
}

func (c *LuciClient) call(ctx context.Context, url string, postBody []byte) ([]byte, error) {
	body := bytes.NewReader(postBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 226 {
		return respBody, c.httpError(resp.StatusCode)
	}

	return respBody, nil
}

func (c *LuciClient) httpError(code int) error {
	if code == 401 {
		return ErrHttpUnauthorized
	}
	if code == 403 {
		return ErrHttpForbidden
	}
	return fmt.Errorf("http status code: %d", code)
}

func (c *LuciClient) rpcWithAuth(ctx context.Context, path, method string, params []string) (string, error) {
	result, err := c.rpc(ctx, path, method, params)
	if err == nil {
		return result, nil
	}

	if err != ErrHttpUnauthorized && err != ErrHttpForbidden {
		return "", err
	}

	if err = c.Auth(ctx); err != nil {
		return "", err
	}

	return c.rpc(ctx, path, method, params)
}

func (c *LuciClient) UCI(ctx context.Context, method string, params []string) (string, error) {
	return c.rpcWithAuth(ctx, uciPath, method, params)
}

func parseString(obj interface{}) (string, error) {
	if obj == nil {
		return "", errors.New("nil object cannot be parsed")
	}

	var result string
	if _, ok := obj.(string); ok {
		result = fmt.Sprintf("%v", obj)
		return result, nil
	}

	jsonBytes, err := json.Marshal(obj)
	if err == nil {
		result = string(jsonBytes)
	}

	return result, err
}

func parseError(obj interface{}) error {
	result, err := parseString(obj)
	if err != nil {
		return err
	}

	return errors.New(result)
}

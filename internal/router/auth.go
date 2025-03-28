// internal/router/auth.go
package router

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io.rd15.tool/pkg/interfaces"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	loginTimeout    = 3 * time.Second
	cookieHeader    = "psp=admin|||2|||0"
	defaultUsername = "admin"
)

type AuthClient struct {
	httpClient *http.Client
	logWriter  interfaces.LogWriter
}

type LoginResult struct {
	Token string `json:"token"`
	Code  int    `json:"code"`
}

type StatusResult struct {
	Code     int `json:"code"`
	Hardware struct {
		SN       string `json:"sn"`
		Platform string `json:"platform"`
	} `json:"hardware"`
}

func NewAuthClient(logWriter interfaces.LogWriter) *AuthClient {
	return &AuthClient{
		httpClient: &http.Client{
			Timeout: loginTimeout,
		},
		logWriter: logWriter,
	}
}

func (c *AuthClient) EnableSSH(ip, stok string) bool {
	if stok == "" || ip == "" {
		c.logWriter.LogWrite("Stok or IP address is empty, something wrong...")
		return false
	}

	commands := []string{
		"'%0Anvram%20set%20ssh_en%3D1'",
		"'%0Anvram%20commit'",
		"'%0Ased%20-i%20's%2Fchannel%3D.*%2Fchannel%3D%22debug%22%2Fg'%20%2Fetc%2Finit.d%2Fdropbear'",
		"'%0A%2Fetc%2Finit.d%2Fdropbear%20start'",
	}

	fmt.Println("Sending requests to IP:", ip)
	fmt.Println("Using STOK:", stok)

	for _, cmd := range commands {
		data := fmt.Sprintf("uid=1234&key=1234%s", cmd)
		urlReq := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/xqsystem/start_binding", ip, stok)

		fmt.Printf("Sending request to URL: %s\n", urlReq)
		fmt.Printf("Request data: %s\n", data)

		resp, err := http.Post(urlReq, "application/x-www-form-urlencoded", bytes.NewBufferString(data))
		if err != nil {
			c.logWriter.LogWrite("Error making login request: " + err.Error())
			return false
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.logWriter.LogWrite(fmt.Sprintf("Error reading response body: %v", err))
			return false
		}
		resp.Body.Close()

		var response Response
		err = json.Unmarshal(body, &response)
		if err != nil {
			c.logWriter.LogWrite(fmt.Sprintf("Error parsing response body: %v", err))
			return false
		}

		c.logWriter.LogWrite(fmt.Sprintf("Response: %s", strconv.Itoa(response.Code)))

		if response.Code != 0 {
			c.logWriter.LogWrite(fmt.Sprintf("Request failed: code is not 0. Response: %v", response))
			return false
		}
	}

	c.logWriter.LogWrite("SSH success enabled!")
	return true
}

func (c *AuthClient) GetRouterInfo(ip string) *ResponseHttp {
	reqUrl := fmt.Sprintf("http://%s/cgi-bin/luci/api/xqsystem/init_info", ip)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		c.logWriter.LogWrite(fmt.Sprintf("Request error: %v", err))
		return &ResponseHttp{}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logWriter.LogWrite(fmt.Sprintf("HTTP error: %v", err))
		return &ResponseHttp{}
	}
	defer resp.Body.Close()

	r, err := ParseResponse(resp.Body)
	if err != nil {
		c.logWriter.LogWrite(fmt.Sprintf("Parse error: %v", err))
		return &ResponseHttp{}
	}

	return r
}

func ParseResponse(r io.Reader) (*ResponseHttp, error) {
	var resp ResponseHttp
	err := json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type ResponseHttp struct {
	Code           int               `json:"code"`
	IsSupportMesh  int               `json:"isSupportMesh"`
	SecAcc         int               `json:"secAcc"`
	Inited         int               `json:"inited"`
	Connect        int               `json:"connect"`
	RouterID       string            `json:"routerId"`
	IPv6           int               `json:"ipv6"`
	ChildRouter    string            `json:"child_router"`
	MeshNodes      []interface{}     `json:"mesh_nodes"`
	Hardware       string            `json:"hardware"`
	Support160M    int               `json:"support160M"`
	MiioVer        string            `json:"miioVer"`
	IsRedmi        int               `json:"isRedmi"`
	RomVersion     string            `json:"romversion"`
	CountryCode    string            `json:"countrycode"`
	IMEI           string            `json:"imei"`
	Modules        map[string]string `json:"modules"`
	ID             string            `json:"id"`
	RouterName     string            `json:"routername"`
	ShowPrivacy    int               `json:"showPrivacy"`
	DisplayName    string            `json:"displayName"`
	MiioDid        string            `json:"miioDid"`
	ModuleVersion  string            `json:"moduleVersion"`
	MAccel         string            `json:"maccel"`
	Model          string            `json:"model"`
	WifiAP         int               `json:"wifi_ap"`
	Bound          int               `json:"bound"`
	NewEncryptMode int               `json:"newEncryptMode"`
	Language       string            `json:"language"`
}

func (c *AuthClient) GetSSHCredentials(ip, password string) (string, string, error) {
	stok, serial, _, err := c.QuerySerial(ip, password)
	if err != nil {
		return "", "", err
	}
	return stok, c.CalcPasswd(serial), nil
}

func (c *AuthClient) QuerySerial(ip, password string) (stok, serial string, model string, err error) {
	token, err := c.authenticate(ip, password)
	if err != nil {
		return "", "", "", fmt.Errorf("authentication failed: %w", err)
	}

	status, err := c.getStatus(ip, token)
	if err != nil {
		return "", "", "", fmt.Errorf("status request failed: %w", err)
	}

	if status.Code != 0 {
		return "", "", "", fmt.Errorf("invalid status code: %d", status.Code)
	}

	return token, status.Hardware.SN, status.Hardware.Platform, nil
}

func (c *AuthClient) RebootRouter(ip, stok string) bool {
	rebootURL := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/xqsystem/reboot", ip, stok)
	req, err := http.NewRequest("GET", rebootURL, nil)
	if err != nil {
		c.logWrite("Error creating reboot request: " + err.Error())
		return false
	}

	c.setCommonHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logWrite("Error making reboot request: " + err.Error())
		return false
	}
	defer resp.Body.Close()
	return true
}

func (c *AuthClient) authenticate(ip, password string) (string, error) {
	loginURL := fmt.Sprintf("http://%s/cgi-bin/luci/api/xqsystem/login", ip)

	data := url.Values{
		"password": {password},
		"logtype":  {"2"},
		"username": {defaultUsername},
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		c.logWrite("Error creating login request: " + err.Error())
		return "", fmt.Errorf("create request failed: %w", err)
	}

	c.setCommonHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logWrite("Error making login request: " + err.Error())
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	var result LoginResult
	if err := c.parseJSONResponse(resp, &result); err != nil {
		c.logWrite("Error parsing login response: " + err.Error())
		return "", fmt.Errorf("parse login response failed: %w", err)
	}

	if result.Code != 0 {
		c.logWrite(fmt.Sprintf("Login failed with code: %d", result.Code))
		return "", fmt.Errorf("login error code: %d", result.Code)
	}

	return result.Token, nil
}

func (c *AuthClient) getStatus(ip, token string) (*StatusResult, error) {
	statusURL := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/misystem/newstatus", ip, token)

	req, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		c.logWrite("Error creating status request: " + err.Error())
		return nil, fmt.Errorf("create status request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logWrite("Error making status request: " + err.Error())
		return nil, fmt.Errorf("status request failed: %w", err)
	}
	defer resp.Body.Close()

	var status StatusResult
	if err := c.parseJSONResponse(resp, &status); err != nil {
		c.logWrite("Error parsing status response: " + err.Error())
		return nil, fmt.Errorf("parse status response failed: %w", err)
	}

	return &status, nil
}

func (c *AuthClient) setCommonHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieHeader)
}

func (c *AuthClient) parseJSONResponse(resp *http.Response, target interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body failed: %w", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}

	return nil
}

func (c *AuthClient) CalcPasswd(sn string) string {
	passwd := sn + getSalt(sn)
	hash := md5.Sum([]byte(passwd))
	password := fmt.Sprintf("%x", hash)[:8]
	return password
}

func getSalt(sn string) string {
	if !strings.Contains(sn, "/") {
		return salt["r1d"]
	}
	parts := strings.Split(salt["others"], "-")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "-")
}

var salt = map[string]string{
	"r1d":    "A2E371B0-B34B-48A5-8C40-A7133F3B5D88",
	"others": "d44fb0960aa0-a5e6-4a30-250f-6d2df50a",
}

func (c *AuthClient) logWrite(message string) {
	if c.logWriter != nil {
		c.logWriter.LogWrite(message)
	}
}

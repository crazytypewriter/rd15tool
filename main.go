package main

import (
	"bytes"
	"crypto/md5"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed sing-box
var embeddedSingBoxBinary []byte

//go:embed singboxini
var embeddedSingBoxIni []byte

type AppWindow struct {
	stokInput          *widget.Entry
	ipInput            *widget.Entry
	passwordInput      *widget.Entry
	sshLoginButton     *widget.Button
	logText            *widget.TextGrid
	logContent         string
	singboxConfigInput *widget.Entry
	sshPasswordLabel   *widget.Label
	sshPasswordInput   *widget.Entry
	routerImage        *canvas.Image
	routerModel        string
	sshEnabled         *widget.Entry
	vlanIdEntry        *widget.Entry
}

type Response struct {
	Code int `json:"code"`
}

func SplitString(input string) string {
	maxLen := 95
	var result string

	for i := 0; i < len(input); i += maxLen {
		end := i + maxLen
		if end > len(input) {
			end = len(input)
		}
		substr := input[i:end]
		if len(substr) > 0 && substr[len(substr)-1] != '\n' {
			substr += "\n"
		}
		result += substr
	}
	return result
}

func (w *AppWindow) enableSSH() bool {
	stok := w.stokInput.Text
	ip := w.ipInput.Text
	if stok == "" || ip == "" {
		w.logContent += "Stok or IP address is empty, something wrong...\n"
		w.logText.SetText(w.logContent)
		return false
	}

	commands := []string{
		"'%0Anvram%20set%20ssh_en%3D1'",
		"'%0Anvram%20commit'",
		"'%0Ased%20-i%20's%2Fchannel%3D.*%2Fchannel%3D%22debug%22%2Fg'%20%2Fetc%2Finit.d%2Fdropbear'",
		"'%0A%2Fetc%2Finit.d%2Fdropbear%20start'",
	}

	// Log the start of the request process
	fmt.Println("Sending requests to IP:", ip)
	fmt.Println("Using STOK:", stok)

	for _, cmd := range commands {
		data := fmt.Sprintf("uid=1234&key=1234%s", cmd)
		urlReq := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/xqsystem/start_binding", ip, stok)

		// Log the request details
		fmt.Printf("Sending request to URL: %s\n", urlReq)
		fmt.Printf("Request data: %s\n", data)

		resp, err := http.Post(urlReq, "application/x-www-form-urlencoded", bytes.NewBufferString(data))
		if err != nil {
			w.logContent += fmt.Sprintf("Error: %v", err)
			w.logText.SetText(w.logContent)
			return false
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			w.logContent += fmt.Sprintf("Error reading response body: %v\n", err)
			w.logText.SetText(w.logContent)
			return false
		}
		resp.Body.Close()

		var response Response
		err = json.Unmarshal(body, &response)
		if err != nil {
			w.logContent += fmt.Sprintf("Error parsing response body: %v\n", err)
			w.logText.SetText(w.logContent)
			return false
		}

		w.logContent += fmt.Sprintf("Response: %s\n", strconv.Itoa(response.Code))
		w.logText.SetText(w.logContent)

		if response.Code != 0 {
			w.logContent += fmt.Sprintf("Request failed: code is not 0. Response: %v\n", response)
			w.logText.SetText(w.logContent)
			return false
		}
	}

	w.logContent += fmt.Sprintf("SSH success enabled!\n")
	w.logText.SetText(w.logContent)
	return true
}

var salt = map[string]string{
	"r1d":    "A2E371B0-B34B-48A5-8C40-A7133F3B5D88",
	"others": "d44fb0960aa0-a5e6-4a30-250f-6d2df50a",
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

func calcPasswd(sn string) string {
	passwd := sn + getSalt(sn)
	hash := md5.Sum([]byte(passwd))
	password := fmt.Sprintf("%x", hash)[:8]
	return password
}

func (w *AppWindow) getSshPassAndStok() error {
	ip := w.ipInput.Text
	routerPassword := w.passwordInput.Text
	sshPass := w.sshPasswordInput.Text

	if ip == "" || routerPassword == "" {
		w.logText.SetText("Please provide both IP address and password.")
		return fmt.Errorf("Please provide both IP address and password")
	}

	if sshPass == "" {
		_, serialNumber, err := w.querySerial(ip, routerPassword)
		if err != nil {
			return err
		}
		sshPass = calcPasswd(serialNumber)

		w.sshPasswordInput.SetText(sshPass)
	}

	return nil
}

func (w *AppWindow) loginSSH(firstTime ...int) (*ssh.Client, error) {
	ip := w.ipInput.Text
	routerPassword := w.passwordInput.Text
	sshPass := w.sshPasswordInput.Text

	if ip == "" || routerPassword == "" {
		w.logText.SetText("Please provide both IP address and password.")
		return nil, fmt.Errorf("Please provide both IP address and password")
	}

	if sshPass == "" {
		_, serialNumber, err := w.querySerial(ip, routerPassword)
		if err != nil {
			return nil, err
		}
		sshPass = calcPasswd(serialNumber)

		if len(firstTime) != 0 {
			w.sshPasswordInput.SetText(sshPass)
		}
		w.enableSSH()
	}

	clientConfig := &ssh.ClientConfig{
		User:              "root", // Adjust the user as necessary
		Auth:              []ssh.AuthMethod{ssh.Password(sshPass)},
		HostKeyCallback:   ssh.InsecureIgnoreHostKey(),
		HostKeyAlgorithms: []string{"ssh-rsa"},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), clientConfig)
	if err != nil {
		w.logContent += SplitString(fmt.Sprintf("Failed SSH connect: %v", err))
		w.logText.SetText(w.logContent)
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		w.logContent += fmt.Sprintf("Failed to create session: %v", err)
		w.logText.SetText(w.logContent)
		return nil, err
	}
	defer session.Close()

	w.LogWrite("SSH login Success!\n")

	return client, nil
}

func (w *AppWindow) enableSSHPermanent() {
	client, err := w.loginSSH()
	if err != nil {
		w.logText.SetText(w.logContent)
	}

	_, err = runSSHCommand(client, "mkdir", "-p", "/etc/crontabs/patches", ">/dev/null 2>&1")
	if err != nil {
		w.logContent += fmt.Sprintf("Failed to create crontabs path directory: %v", err)
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Check path Success!\n")
	w.logText.SetText(w.logContent)

	err = copyFileToRemote(client, "ssh_patch.sh", "/etc/crontabs/patches/ssh_patch.sh") // Adjust paths as needed
	if err != nil {
		fmt.Printf("Failed to copy file to remote server: %v\n", err)
		return
	}
	w.logText.SetText(fmt.Sprintf("SSH patch installed to disk!\n"))

	cmdR := "crontab -l > /tmp/current_crontab && if ! grep -q 'ssh_patch.sh' /tmp/current_crontab; then echo '*/1 * * * * /etc/crontabs/patches/ssh_patch.sh >/dev/null 2>&1' >> /tmp/current_crontab; crontab /tmp/current_crontab; fi"
	_, err = runSSHCommand(client, cmdR)
	if err != nil {
		w.logContent += fmt.Sprintf("Failed to add SSH check to cron: %v\n", err)
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("SSH installed!\n")
	w.logText.SetText(w.logContent)

	_, err = runSSHCommand(client, "/etc/init.d/cron restart")
	if err != nil {
		w.logContent += fmt.Sprintf("Cron restarted error: %s\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Cron restarted successfully!\n")
	w.logText.SetText(w.logContent)

	// Log success
	w.logContent += fmt.Sprintf("SSH login and script copied successfully.\n")
	w.logText.SetText(w.logContent)
}

func (w *AppWindow) enableSingboxPermanent() {
	client, err := w.loginSSH()
	if err != nil {
		w.logText.SetText(w.logContent)
	}

	_, err = runSSHCommand(client, "mkdir", "-p", "/etc/crontabs/patches", ">/dev/null 2>&1")
	if err != nil {
		w.logContent += fmt.Sprintf("Failed to create crontabs path directory: %v", err)
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Check path Success!\n")
	w.logText.SetText(w.logContent)

	err = copyFileToRemote(client, "singbox_patch.sh", "/etc/crontabs/patches/singbox_patch.sh") // Adjust paths as needed
	if err != nil {
		fmt.Printf("Failed to copy file to remote server: %v\n", err)
		return
	}
	w.logText.SetText(fmt.Sprintf("Sing-box patch installed to disk!\n"))

	cmdR := "crontab -l > /tmp/current_crontab && if ! grep -q 'singbox_patch.sh' /tmp/current_crontab; then echo '*/1 * * * * /etc/crontabs/patches/singbox_patch.sh >/dev/null 2>&1' >> /tmp/current_crontab; crontab /tmp/current_crontab; fi"
	_, err = runSSHCommand(client, cmdR)
	if err != nil {
		w.logContent += fmt.Sprintf("Failed to add SSH check to cron: %v\n", err)
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box installed!\n")
	w.logText.SetText(w.logContent)

	_, err = runSSHCommand(client, "/etc/init.d/cron restart")
	if err != nil {
		w.logContent += fmt.Sprintf("Cron restarted error: %s\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Cron restarted successfully!\n")
	w.logText.SetText(w.logContent)

	w.logContent += fmt.Sprintf("Sing-box and script copied successfully.\n")
	w.logText.SetText(w.logContent)
}

func (w *AppWindow) LogWrite(message string) {
	w.logContent += fmt.Sprintf(SplitString(message))
	w.logText.SetText(w.logContent)
}

func runSSHCommand(client *ssh.Client, args ...string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = os.Stderr
	cmd := exec.Command(args[0], args[1:]...)
	if err := session.Run(cmd.String()); err != nil {
		return stdoutBuf.String(), fmt.Errorf("failed to execute command: %w", err)
	}
	return stdoutBuf.String(), nil
}

func copyFileToRemote(client *ssh.Client, localPath, remotePath string) error {
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer srcFile.Close()

	srcFileInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	pipe, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	defer pipe.Close()

	cmd := fmt.Sprintf("scp -t %s", remotePath)
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("failed to start remote scp command: %w", err)
	}

	fmt.Fprintf(pipe, "C%#o %d %s\n", srcFileInfo.Mode().Perm()|0111, srcFileInfo.Size(), filepath.Base(remotePath))

	if _, err := io.Copy(pipe, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	fmt.Fprint(pipe, "\x00")
	pipe.Close()

	if err := session.Wait(); err != nil {
		return fmt.Errorf("failed to complete scp command: %w", err)
	}

	return nil
}

func getLocalSubnet() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if ok && ipNet.IP.To4() != nil {
					ip := ipNet.IP.String()
					subnet := strings.Join(strings.Split(ip, ".")[:3], ".")
					return subnet, nil
				}
			}
		}
	}
	return "", fmt.Errorf("can't detect network")
}

func checkSSHPort(ip string, port int, wg *sync.WaitGroup, resultChan chan<- string) {
	defer wg.Done()

	address := fmt.Sprintf("%s:%d", ip, port)

	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()

	resp, err := http.Get(fmt.Sprintf("http://%s:%d", ip, port))
	if err != nil {
		fmt.Printf("Failed to connect to %s:%d\n", ip, port)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body from %s:%d\n", ip, port)
		return
	}

	re := regexp.MustCompile(`<p class="rom-ver">.*?: ([^<]+).*?MAC.*?: ([^<]+)</p>`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) > 0 {
		fmt.Printf("Port 8099 open on %s\nFirmware Version: %s\nMAC Address: %s\n", ip, matches[1], matches[2])
		resultChan <- ip
	} else {
		fmt.Printf("Port 8099 open on %s, but failed to extract version and MAC\n", ip)
	}
}

func checkPort(ip string, port int, wg *sync.WaitGroup, resultChan chan<- string) {
	defer wg.Done()

	address := fmt.Sprintf("%s:%d", ip, port)

	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()

	resp, err := http.Get(fmt.Sprintf("http://%s:%d", ip, port))
	if err != nil {
		fmt.Printf("Failed to connect to %s:%d\n", ip, port)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body from %s:%d\n", ip, port)
		return
	}

	re := regexp.MustCompile(`<p class="rom-ver">.*?: ([^<]+).*?MAC.*?: ([^<]+)</p>`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) > 0 {
		fmt.Printf("Port 8099 open on %s\nFirmware Version: %s\nMAC Address: %s\n", ip, matches[1], matches[2])
		resultChan <- ip
	} else {
		fmt.Printf("Port 8099 open on %s, but failed to extract version and MAC\n", ip)
	}
}

func (w *AppWindow) getCheckSSHEnabled() bool {
	client, err := w.loginSSH()
	if err != nil {
		w.logText.SetText(w.logContent)
	}

	banner, err := runSSHCommand(client, "cat", "/etc/banner")
	if err != nil {
		w.logContent += fmt.Sprintf("Failed to get ssh welcome: %v", err)
		w.logText.SetText(w.logContent)
	}
	w.logText.SetText(banner)
	return true

}

func detectRouterIP(ipInput *widget.Entry) {
	subnet, err := getLocalSubnet()
	if err != nil {
		fmt.Println("Error getting subnet:", err)
		return
	}

	fmt.Printf("Scanning subnet: %s.0/24\n", subnet)

	var wg sync.WaitGroup
	port := 8099
	resultChan := make(chan string, 1) // Buffer size of 1 to get the first valid IP

	for i := 1; i <= 254; i++ {
		ip := fmt.Sprintf("%s.%d", subnet, i)
		wg.Add(1)
		go checkPort(ip, port, &wg, resultChan)
	}

	// Wait for either a valid IP or the completion of all goroutines
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	select {
	case foundIP := <-resultChan:
		fmt.Printf("Found IP: %s\n", foundIP)
		ipInput.SetText(foundIP) // Update the IP input field
	case <-time.After(30 * time.Second): // Timeout after 30 seconds
		fmt.Println("No valid IP found in the subnet.")
	}
}

func (w *AppWindow) querySerial(ip, pass string) (stok string, serial string, err error) {
	loginURL := fmt.Sprintf("http://%s/cgi-bin/luci/api/xqsystem/login", ip)
	loginData := url.Values{
		"password": {pass},
		"logtype":  {"2"},
		"username": {"admin"},
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBufferString(loginData.Encode()))
	if err != nil {
		w.LogWrite(fmt.Sprintf("Error creating login request: %s", err))
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "psp=admin|||2|||0")

	client := &http.Client{}
	client.Timeout = 3 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		w.LogWrite(fmt.Sprintf("Error making login request: %s", err))
		return "", "", err
	}
	defer resp.Body.Close()
	w.LogWrite("Try http login to router\n")

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.LogWrite(fmt.Sprintf("Error reading login response body: %s", err))
		return "", "", err
	}
	w.LogWrite("Read http answer\n")

	var loginResult struct {
		Token string `json:"token"`
		Code  int    `json:"code"`
	}
	err = json.Unmarshal(body, &loginResult)
	if err != nil {
		w.LogWrite(fmt.Sprintf("Error parsing login response body: %s", err))
		return "", "", err
	}
	w.LogWrite("Unmarshal answer\n")

	if loginResult.Code != 0 {
		w.LogWrite(fmt.Sprintf("Login failed with code: %d", loginResult.Code))
		return "", "", fmt.Errorf("Login failed with code: %d", loginResult.Code)
	}

	w.stokInput.SetText(loginResult.Token)
	w.LogWrite("Stok received\n")

	stok = loginResult.Token
	statusURL := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/misystem/newstatus", ip, stok)

	req, err = http.NewRequest("GET", statusURL, nil)
	if err != nil {
		w.LogWrite(fmt.Sprintf("Error creating status request: %s", err))
		return "", "", err
	}

	resp, err = client.Do(req)
	if err != nil {
		w.LogWrite(fmt.Sprintf("Error making status request: %s", err))
		return "", "", err
	}
	defer resp.Body.Close()
	w.LogWrite("Make http query to router\n")

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading status response body:", err)
		return "", "", err
	}
	w.LogWrite("Read answer\n")

	var statusResult struct {
		Code     int `json:"code"`
		Hardware struct {
			SN       string `json:"sn"`
			Platform string `json:"platform"`
		} `json:"hardware"`
	}
	err = json.Unmarshal(body, &statusResult)
	if err != nil {
		fmt.Println("Error parsing status JSON:", err)
		return "", "", err
	}
	w.routerModel = statusResult.Hardware.Platform
	//fmt.Println(string(body[:]))
	w.LogWrite("Unmarshal answer\n")

	if statusResult.Code == 0 {
		w.LogWrite("Request successful!")
		fmt.Println("SN value:", statusResult.Hardware.SN)
	} else {
		w.LogWrite(fmt.Sprintf("Request failed with code: %d", statusResult.Code))
		return "", "", err
	}
	return stok, statusResult.Hardware.SN, nil
}

func (w *AppWindow) showProgressWithDots(startText string, updateText func(string), task func() error) {
	w.logContent += startText
	updateText(w.logContent)

	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		dots := "."

		for {
			select {
			case <-ticker.C:
				dots += "."
				updateText(w.logContent + dots)
			case <-done:
				return
			}
		}
	}()

	err := task()
	close(done)

	if err != nil {
		w.logContent += fmt.Sprintf("\nError: %s.\n", err.Error())
	} else {
		w.logContent += " success!\n"
	}
	updateText(w.logContent)
}

func (w *AppWindow) installSingBox() {
	updateText := func(text string) {
		w.logText.SetText(SplitString(text))
	}

	err := writeToFile("sing-box_temp", embeddedSingBoxBinary)
	if err != nil {
		w.logContent += fmt.Sprintf("Sing-box file write to local disk error %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box file write to local disk success!.\n")
	w.logText.SetText(w.logContent)

	client, err := w.loginSSH()

	if err != nil {
		w.logContent += fmt.Sprintf("Error ssh login %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}

	_, err = runSSHCommand(client, "mkdir", "-p", "/data/etc/sing-box")
	if err != nil {
		w.logContent += fmt.Sprintf("Error mkdir for sing-box %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box mkdir success!.\n")
	w.logText.SetText(w.logContent)

	copyTask := func() error {
		return copyFileToRemote(client, "./sing-box_temp", "/data/etc/sing-box/sing-box")
	}
	w.showProgressWithDots("Trying to copy binary file to router", updateText, copyTask)

	err = writeToFile("singboxini", embeddedSingBoxIni)
	if err != nil {
		w.logContent += fmt.Sprintf("Sing-box file write to local disk error %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box file write to local disk success!.\n")
	w.logText.SetText(w.logContent)

	copyInitTask := func() error {
		return copyFileToRemote(client, "./singboxini", "/etc/init.d/sing-box")
	}
	w.showProgressWithDots("Sing-box init file copied to router", updateText, copyInitTask)
}

func (w *AppWindow) installSingBoxConfig() {
	client, err := w.loginSSH()

	path := w.singboxConfigInput.Text
	if strings.HasPrefix(path, "file://") {
		path = strings.TrimPrefix(path, "file://")
	}
	err = copyFileToRemote(client, path, "/data/etc/sing-box/config.json")
	if err != nil {
		w.logContent += fmt.Sprintf("Error copying config file to router %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box config file copied to router success!.\n")
	w.logText.SetText(w.logContent)
}

func (w *AppWindow) startSingBox() {
	client, err := w.loginSSH()

	_, err = runSSHCommand(client, "/etc/init.d/sing-box", "start")
	if err != nil {
		w.logContent += fmt.Sprintf("Error starting sing-box %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box start successful!.\n")
	w.logText.SetText(w.logContent)
}

func (w *AppWindow) checkSingBoxStarted() {
	client, err := w.loginSSH()

	res, err := runSSHCommand(client, "/data/etc/sing-box/sing-box check -c /data/etc/sing-box/config.json -D /tmp/sing-box/")
	if err != nil {
		w.logContent += fmt.Sprintf("Result %s, Error %s.\n", res, err.Error())
		w.logText.SetText(w.logContent)
	}
	if res != "" {
		w.logContent += fmt.Sprintf("Sing-box start check %v.\n", res)
		w.logText.SetText(w.logContent)
	}

}

func (w *AppWindow) stopSingBox() {
	client, err := w.loginSSH()

	_, err = runSSHCommand(client, "/etc/init.d/sing-box", "stop")
	if err != nil {
		w.logContent += fmt.Sprintf("Error stopping sing-box %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box stop successful!.\n")
	w.logText.SetText(w.logContent)
}

func (w *AppWindow) replaceRemoteFileRegex(filePath string, replacements map[*regexp.Regexp]string) error {
	client, err := w.loginSSH()
	if err != nil {
		return err
	}

	catCommand := fmt.Sprintf("cat %s", filePath)
	fileContent, err := runSSHCommand(client, catCommand)
	if err != nil {
		w.LogWrite(fmt.Sprintf("Error running command: %s.\n", err.Error()))
		return fmt.Errorf("failed to read file: %w", err)
	}

	modified := false
	lines := strings.Split(fileContent, "\n")
	for i, line := range lines {
		for re, replacement := range replacements {
			if re.MatchString(line) {
				newLine := re.ReplaceAllString(line, replacement)
				if line != newLine {
					lines[i] = newLine
					modified = true
				}
			}
		}
	}

	if !modified {
		fmt.Println("No changes needed for", filePath)
		w.LogWrite(fmt.Sprintf("No changes needed for %s", filePath))
		return nil
	}

	updatedContent := strings.Join(lines, "\n")

	echoCommand := fmt.Sprintf("echo -e %q > %s", updatedContent, filePath)
	if _, err = runSSHCommand(client, echoCommand); err != nil {
		w.LogWrite(fmt.Sprintf("failed to update file: %w", err))
		return fmt.Errorf("failed to update file: %w", err)
	}

	w.LogWrite("File updated successfully on remote host!\n")
	return nil
}

func (w *AppWindow) enableVLAN() {
	id := w.vlanIdEntry.Text
	newReplacement := "." + id
	if id == "0" {
		newReplacement = ""
	}

	fileModifications := map[string]map[string]string{
		"/etc/config/network": {
			`config interface 'eth1(\.\d+)?'`:      "config interface 'eth1" + newReplacement + "'",
			`option ifname '([^']*?)eth1(\.\d+)?'`: "option ifname '${1}eth1" + newReplacement + "'",
		},
		"/etc/config/port_map": {
			`option ifname 'eth1(\.\d+)?'`: "option ifname 'eth1" + newReplacement + "'",
		},
	}

	for filePath, patterns := range fileModifications {
		replacements := make(map[*regexp.Regexp]string)
		for pattern, replacement := range patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Println("Error compiling regex:", err)
				w.LogWrite(fmt.Sprintf("Error compiling regex: %s.\n", err.Error()))
				return
			}
			replacements[re] = replacement
		}

		if err := w.replaceRemoteFileRegex(filePath, replacements); err != nil {
			fmt.Println("Error updating", filePath, "on remote host:", err)
			w.LogWrite(fmt.Sprintf("Error updating: %s", err.Error()))
		}
	}
}

func (w *AppWindow) enableUART() {
	fileModifications := map[string]map[string]string{
		"/etc/inittab": {
			`ttyMSM0::askfirst:/bin/ash\s+--login`: "ttyMSM0::askfirst:/bin/ash", // Updated regex pattern
		},
	}

	for filePath, patterns := range fileModifications {
		replacements := make(map[*regexp.Regexp]string)
		for pattern, replacement := range patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Println("Error compiling regex:", err)
				w.LogWrite(fmt.Sprintf("Error compiling regex: %s.\n", err.Error()))
				return
			}
			replacements[re] = replacement
		}

		if err := w.replaceRemoteFileRegex(filePath, replacements); err != nil {
			fmt.Println("Error updating", filePath, "on remote host:", err)
			w.LogWrite(fmt.Sprintf("Error updating: %s", err.Error()))
		}
	}
}

func writeToFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("RD15 Tool")
	myWindow.Resize(fyne.NewSize(800, 1200))
	myWindow.CenterOnScreen()
	sshEnabled := widget.NewEntry()

	ipLabel := widget.NewLabel("IP Address:")
	ipInput := widget.NewEntry()

	passwordLabel := widget.NewLabel("Password:                                                       ")
	passwordInput := widget.NewPasswordEntry()
	passwordInput.SetPlaceHolder("First time setup router password")

	routerImage := canvas.NewImageFromFile("be3600.png")
	routerImage.SetMinSize(fyne.NewSize(75, 75))
	routerImage.FillMode = canvas.ImageFillContain
	routerImage.Hide()

	passwordContainer := container.NewHBox(
		container.NewVBox(passwordLabel, passwordInput),
		layout.NewSpacer(),
		routerImage,
	)

	stokLabel := widget.NewLabel("Stok (get automatic):")
	stokInput := widget.NewEntry()
	stokInput.Disable()

	stokInputClipBoardButton := widget.NewButtonWithIcon("copy", theme.ContentCopyIcon(), func() {
		myWindow.Clipboard().SetContent(stokInput.Text)
	})

	stokInputBorder := container.NewBorder(nil, nil, nil, stokInputClipBoardButton, stokInput)

	sshPasswordLabel := widget.NewLabel("SSH Password (calculated automatic):")
	sshPasswordInput := widget.NewEntry()
	sshPasswordInput.Disable()

	sshPasswordInputClipBoardButton := widget.NewButtonWithIcon("copy", theme.ContentCopyIcon(), func() {
		myWindow.Clipboard().SetContent(sshPasswordInput.Text)
	})

	sshPasswordBorder := container.NewBorder(nil, nil, nil, sshPasswordInputClipBoardButton, sshPasswordInput)

	singboxConfigInput := widget.NewEntry()

	openFileButton := widget.NewButton("Choose singbox config", func() {
		fileDialog := dialog.NewFileOpen(
			func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, myWindow)
					return
				}
				if reader == nil {
					return
				}
				singboxConfigInput.SetText(reader.URI().String())
				defer reader.Close()
			}, myWindow)

		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
		fileDialog.Show()
	})
	openFileButton.Disable()

	logText := widget.NewTextGrid()
	logText.SetText("")

	go detectRouterIP(ipInput)

	trySSHLoginButton := widget.NewButton("Get SSH password", func() {
		appWindow := &AppWindow{
			stokInput:        stokInput,
			ipInput:          ipInput,
			passwordInput:    passwordInput,
			logText:          logText,
			sshPasswordLabel: sshPasswordLabel,
			sshPasswordInput: sshPasswordInput,
		}
		err := appWindow.getSshPassAndStok()
		if err != nil {
			return
		}

		if appWindow.routerModel != "" {
			var pic string
			if pic, err = getPicForRouter(appWindow.routerModel); err == nil {
				routerImage.File = pic
				routerImage.Refresh()
				routerImage.Show()
			}

		}

		if appWindow.getCheckSSHEnabled() {
			sshEnabled.SetText("SSH enabled")
		}
	})

	sshEnableButton := widget.NewButton("Enable SSH", func() {
		appWindow := &AppWindow{
			stokInput:        stokInput,
			ipInput:          ipInput,
			passwordInput:    passwordInput,
			logText:          logText,
			sshPasswordLabel: sshPasswordLabel,
			sshPasswordInput: sshPasswordInput,
		}
		if appWindow.enableSSH() {
			sshEnabled.SetText("SSH enabled")
		}
	})
	sshEnableButton.Disable()

	sshLoginButton := widget.NewButton("Enable SSH permanent", func() {
		appWindow := &AppWindow{
			stokInput:        stokInput,
			ipInput:          ipInput,
			passwordInput:    passwordInput,
			logText:          logText,
			sshPasswordInput: sshPasswordInput,
		}
		appWindow.enableSSHPermanent()
	})
	sshLoginButton.Disable()

	installSingBox := widget.NewButton("Install sing-box", func() {
		appWindow := &AppWindow{
			stokInput:          stokInput,
			ipInput:            ipInput,
			passwordInput:      passwordInput,
			logText:            logText,
			singboxConfigInput: singboxConfigInput,
			sshPasswordInput:   sshPasswordInput,
		}
		appWindow.installSingBox()
	})
	installSingBox.Disable()

	installSingBoxConfig := widget.NewButton("Install sing-box config file", func() {
		appWindow := &AppWindow{
			stokInput:          stokInput,
			ipInput:            ipInput,
			passwordInput:      passwordInput,
			logText:            logText,
			singboxConfigInput: singboxConfigInput,
			sshPasswordInput:   sshPasswordInput,
		}
		appWindow.installSingBoxConfig()
	})
	installSingBoxConfig.Disable()

	startSingBox := widget.NewButton("Start SingBox", func() {
		appWindow := &AppWindow{
			stokInput:          stokInput,
			ipInput:            ipInput,
			passwordInput:      passwordInput,
			logText:            logText,
			singboxConfigInput: singboxConfigInput,
			sshPasswordInput:   sshPasswordInput,
		}
		appWindow.startSingBox()

		appWindow.checkSingBoxStarted()
	})
	startSingBox.Disable()

	stopSingBox := widget.NewButton("Stop SingBox", func() {
		appWindow := &AppWindow{
			stokInput:          stokInput,
			ipInput:            ipInput,
			passwordInput:      passwordInput,
			logText:            logText,
			singboxConfigInput: singboxConfigInput,
			sshPasswordInput:   sshPasswordInput,
		}
		appWindow.stopSingBox()
	})
	stopSingBox.Disable()

	enableSingboxPermanent := widget.NewButton("Enable Sing-box Permanent", func() {
		appWindow := &AppWindow{
			stokInput:        stokInput,
			ipInput:          ipInput,
			passwordInput:    passwordInput,
			logText:          logText,
			sshPasswordInput: sshPasswordInput,
		}
		appWindow.enableSingboxPermanent()
	})
	enableSingboxPermanent.Disable()

	vlanIdEntry := widget.NewEntry()
	enableVLAN := widget.NewButton("Set VLAN on port 4", func() {
		appWindow := &AppWindow{
			stokInput:        stokInput,
			ipInput:          ipInput,
			passwordInput:    passwordInput,
			logText:          logText,
			sshPasswordInput: sshPasswordInput,
			vlanIdEntry:      vlanIdEntry,
		}
		appWindow.enableVLAN()
	})
	enableVLAN.Disable()
	//vlanBorder := container.NewBorder(nil, nil, nil, enableVLAN, vlanIdEntry)

	sshPasswordInput.OnChanged = func(text string) {
		if len(text) > 5 {
			sshEnableButton.Enable()
		}
	}

	enableUART := widget.NewButton("Set UART on", func() {
		appWindow := &AppWindow{
			stokInput:        stokInput,
			ipInput:          ipInput,
			passwordInput:    passwordInput,
			logText:          logText,
			sshPasswordInput: sshPasswordInput,
			vlanIdEntry:      vlanIdEntry,
		}
		appWindow.enableUART()
	})
	enableUART.Disable()

	sshEnabled.OnChanged = func(text string) {
		if len(text) > 0 {
			enableSingboxPermanent.Enable()
			stopSingBox.Enable()
			startSingBox.Enable()
			installSingBoxConfig.Enable()
			sshLoginButton.Enable()
			installSingBox.Enable()
			openFileButton.Enable()
			enableUART.Enable()
		}
	}

	content := container.NewVBox(
		ipLabel, ipInput,
		passwordContainer,
		stokLabel, stokInputBorder,
		sshPasswordLabel, sshPasswordBorder,
		trySSHLoginButton, sshEnableButton, sshLoginButton, openFileButton, singboxConfigInput, installSingBox, installSingBoxConfig, startSingBox, stopSingBox, enableSingboxPermanent,
		//vlanBorder,
		enableUART,
		logText,
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

func getPicForRouter(model string) (string, error) {
	switch model {
	case "RD15":
		return "./be3600.png", nil
	case "RD16":
		return "./be5000.png", nil
	}
	return "", errors.New("invalid model")
}

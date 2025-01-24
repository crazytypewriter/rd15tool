package main

import (
	"bytes"
	"crypto/md5"
	_ "embed"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
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
	"strings"
	"sync"
	"time"
)

//go:embed sing-box
var embeddedFile []byte

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
}

type Response struct {
	Code int `json:"code"`
}

func (w *AppWindow) enableSSH() {
	stok := w.stokInput.Text
	ip := w.ipInput.Text
	if stok == "" || ip == "" {
		w.logContent += "Please fill in both stok and IP address.\n"
		w.logText.SetText(w.logContent)
		return
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
		data := fmt.Sprintf("uid=1234&key=1234'%s'", cmd)
		urlReq := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/xqsystem/start_binding", ip, stok)

		// Log the request details
		fmt.Printf("Sending request to URL: %s\n", urlReq)
		fmt.Printf("Request data: %s\n", data)

		w.logContent += fmt.Sprintf("Sending request to URL: %s\n", urlReq)
		w.logText.SetText(w.logContent)
		w.logContent += fmt.Sprintf("Request data: %s\n", data)
		w.logText.SetText(w.logContent)

		resp, err := http.Post(urlReq, "application/x-www-form-urlencoded", bytes.NewBufferString(data))
		if err != nil {
			w.logContent += fmt.Sprintf("Error: %v", err)
			w.logText.SetText(w.logContent)
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			w.logContent += fmt.Sprintf("Error reading response body: %v\n", err)
			w.logText.SetText(w.logContent)
			return
		}
		resp.Body.Close()

		w.logContent += fmt.Sprintf("Response: %s\n", string(body))
		w.logText.SetText(w.logContent)

		var response Response
		err = json.Unmarshal(body, &response)
		if err != nil {
			w.logContent += fmt.Sprintf("Error parsing response body: %v\n", err)
			w.logText.SetText(w.logContent)
			return
		}

		if response.Code != 0 {
			w.logContent += fmt.Sprintf("Request failed: code is not 0. Response: %v\n", response)
			w.logText.SetText(w.logContent)
			return
		}
	}

	// Enable SSH login button after all requests are successful
	//w.sshLoginButton.Hidden = false
}

var salt = map[string]string{
	"r1d":    "A2E371B0-B34B-48A5-8C40-A7133F3B5D88",
	"others": "d44fb0960aa0-a5e6-4a30-250f-6d2df50a",
}

func getSalt(sn string) string {
	if !strings.Contains(sn, "/") {
		return salt["r1d"]
	}
	// Разворачиваем соль для других устройств
	parts := strings.Split(salt["others"], "-")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "-")
}

func calcPasswd(sn string) string {
	passwd := sn + getSalt(sn)
	hash := md5.Sum([]byte(passwd))
	return fmt.Sprintf("%x", hash)[:8]
}

func (w *AppWindow) loginSSH() (*ssh.Client, error) {
	ip := w.ipInput.Text
	routerPassword := w.passwordInput.Text

	_, serialNumber := w.query(ip, routerPassword)
	password := calcPasswd(serialNumber)

	if ip == "" || password == "" {
		w.logText.SetText("Please provide both IP address and password.")
		return nil, nil
	}

	clientConfig := &ssh.ClientConfig{
		User:              "root", // Adjust the user as necessary
		Auth:              []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback:   ssh.InsecureIgnoreHostKey(),
		HostKeyAlgorithms: []string{"ssh-rsa"},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), clientConfig)
	if err != nil {
		w.logContent += fmt.Sprintf("Failed SSH connect: %v", err)
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
	w.sshPasswordInput.SetText(password)
	w.logContent += fmt.Sprintf("SSH login Success!\n")
	w.logText.SetText(w.logContent)

	return client, nil
}

func (w *AppWindow) enableSSHPermanent() {
	client, err := w.loginSSH()
	if err != nil {
		w.logText.SetText(w.logContent)
	}

	err = runSSHCommand(client, "mkdir", "-p", "/etc/crontabs/patches", ">/dev/null 2>&1")
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
	err = runSSHCommand(client, cmdR)
	if err != nil {
		w.logContent += fmt.Sprintf("Failed to add SSH check to cron: %v\n", err)
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("SSH installed!\n")
	w.logText.SetText(w.logContent)

	err = runSSHCommand(client, "/etc/init.d/cron restart")
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

	err = runSSHCommand(client, "mkdir", "-p", "/etc/crontabs/patches", ">/dev/null 2>&1")
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
	err = runSSHCommand(client, cmdR)
	if err != nil {
		w.logContent += fmt.Sprintf("Failed to add SSH check to cron: %v\n", err)
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box installed!\n")
	w.logText.SetText(w.logContent)

	err = runSSHCommand(client, "/etc/init.d/cron restart")
	if err != nil {
		w.logContent += fmt.Sprintf("Cron restarted error: %s\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Cron restarted successfully!\n")
	w.logText.SetText(w.logContent)

	w.logContent += fmt.Sprintf("Sing-box and script copied successfully.\n")
	w.logText.SetText(w.logContent)
}

func runSSHCommand(client *ssh.Client, args ...string) error {
	session, err := client.NewSession()
	cmd := exec.Command(args[0], args[1:]...)
	fmt.Printf("Executing command: %s\n", cmd)
	err = session.Run(cmd.String())
	if err != nil {
		return err
	}
	defer session.Close()
	return nil
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
	return "", fmt.Errorf("не удалось определить подсеть")
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

func (w *AppWindow) query(ip, pass string) (stok string, serial string) {
	loginURL := fmt.Sprintf("http://%s/cgi-bin/luci/api/xqsystem/login", ip)
	loginData := url.Values{
		"password": {pass},
		"logtype":  {"2"},
		"username": {"admin"},
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBufferString(loginData.Encode()))
	if err != nil {
		fmt.Println("Error creating login request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "psp=admin|||2|||0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making login request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading login response body:", err)
		return
	}

	var loginResult struct {
		Token string `json:"token"`
		Code  int    `json:"code"`
	}
	err = json.Unmarshal(body, &loginResult)
	if err != nil {
		fmt.Println("Error parsing login JSON:", err)
		return
	}

	if loginResult.Code != 0 {
		fmt.Println("Login failed with code:", loginResult.Code)
		return
	}

	w.stokInput.SetText(loginResult.Token)

	stok = loginResult.Token
	statusURL := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/misystem/newstatus", ip, stok)

	req, err = http.NewRequest("GET", statusURL, nil)
	if err != nil {
		fmt.Println("Error creating status request:", err)
		return
	}

	resp, err = client.Do(req)
	if err != nil {
		fmt.Println("Error making status request:", err)
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading status response body:", err)
		return
	}

	var statusResult struct {
		Code     int `json:"code"`
		Hardware struct {
			SN string `json:"sn"`
		} `json:"hardware"`
	}
	err = json.Unmarshal(body, &statusResult)
	if err != nil {
		fmt.Println("Error parsing status JSON:", err)
		return
	}

	if statusResult.Code == 0 {
		fmt.Println("Request successful!")
		fmt.Println("SN value:", statusResult.Hardware.SN)
	} else {
		fmt.Println("Request failed with code:", statusResult.Code)
	}
	return stok, statusResult.Hardware.SN
}

func (w *AppWindow) installSingBox() {
	err := writeToFile("sing-box_temp", embeddedFile)
	if err != nil {
		w.logContent += fmt.Sprintf("Sing-box file write to local disk error %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box file write to local disk success!.\n")
	w.logText.SetText(w.logContent)

	client, err := w.loginSSH()

	err = runSSHCommand(client, "mkdir", "-p", "/data/etc/sing-box")
	if err != nil {
		w.logContent += fmt.Sprintf("Error mkdir for sing-box %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box mkdir success!.\n")
	w.logText.SetText(w.logContent)

	err = copyFileToRemote(client, "./sing-box_temp", "/data/etc/sing-box/sing-box")
	if err != nil {
		w.logContent += fmt.Sprintf("Error copying binary file to router %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box binary file copied to remote disk success!.\n")
	w.logText.SetText(w.logContent)

	err = copyFileToRemote(client, "singbox.init", "/etc/init.d/sing-box")
	if err != nil {
		w.logContent += fmt.Sprintf("Error copying init file to router %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box init file copied to router success!.\n")
	w.logText.SetText(w.logContent)

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

	err = runSSHCommand(client, "/etc/init.d/sing-box", "start")
	if err != nil {
		w.logContent += fmt.Sprintf("Error starting sing-box %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box start successful!.\n")
	w.logText.SetText(w.logContent)
}

func (w *AppWindow) stopSingBox() {
	client, err := w.loginSSH()

	err = runSSHCommand(client, "/etc/init.d/sing-box", "stop")
	if err != nil {
		w.logContent += fmt.Sprintf("Error stopping sing-box %s.\n", err.Error())
		w.logText.SetText(w.logContent)
	}
	w.logContent += fmt.Sprintf("Sing-box stop successful!.\n")
	w.logText.SetText(w.logContent)
}

func writeToFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("RD15 Tool")
	myWindow.Resize(fyne.NewSize(800, 800))
	myWindow.CenterOnScreen()

	stokLabel := widget.NewLabel("stok:")
	stokInput := widget.NewEntry()
	stokInput.Disable()

	ipLabel := widget.NewLabel("IP Address:")
	ipInput := widget.NewEntry()

	passwordLabel := widget.NewLabel("Password:")
	passwordInput := widget.NewEntry()
	passwordInput.SetText("")

	sshPasswordLabel := widget.NewLabel("SSH Password:")
	sshPasswordInput := widget.NewEntry()
	sshPasswordInput.Disable()

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

	logText := widget.NewTextGrid()
	logText.SetText("")

	go detectRouterIP(ipInput)

	sendButton := widget.NewButton("Enable SSH", func() {
		appWindow := &AppWindow{
			stokInput:      stokInput,
			ipInput:        ipInput,
			passwordInput:  passwordInput,
			sshLoginButton: nil,
			logText:        logText,
		}
		appWindow.enableSSH()
	})

	sshLoginButton := widget.NewButton("Enable SSH permanent", func() {
		appWindow := &AppWindow{
			stokInput:     stokInput,
			ipInput:       ipInput,
			passwordInput: passwordInput,
			logText:       logText,
		}
		appWindow.enableSSHPermanent()
	})

	installSingBox := widget.NewButton("Install sing-box", func() {
		appWindow := &AppWindow{
			stokInput:          stokInput,
			ipInput:            ipInput,
			passwordInput:      passwordInput,
			logText:            logText,
			singboxConfigInput: singboxConfigInput,
		}
		appWindow.installSingBox()
	})

	trySSHLoginButton := widget.NewButton("Try SSH Login", func() {
		appWindow := &AppWindow{
			stokInput:        stokInput,
			ipInput:          ipInput,
			passwordInput:    passwordInput,
			logText:          logText,
			sshPasswordLabel: sshPasswordLabel,
			sshPasswordInput: sshPasswordInput,
		}
		appWindow.loginSSH()
	})

	installSingBoxConfig := widget.NewButton("Install sing-box config file", func() {
		appWindow := &AppWindow{
			stokInput:          stokInput,
			ipInput:            ipInput,
			passwordInput:      passwordInput,
			logText:            logText,
			singboxConfigInput: singboxConfigInput,
		}
		appWindow.installSingBoxConfig()
	})

	startSingBox := widget.NewButton("Start SingBox", func() {
		appWindow := &AppWindow{
			stokInput:          stokInput,
			ipInput:            ipInput,
			passwordInput:      passwordInput,
			logText:            logText,
			singboxConfigInput: singboxConfigInput,
		}
		appWindow.startSingBox()
	})

	stopSingBox := widget.NewButton("Stop SingBox", func() {
		appWindow := &AppWindow{
			stokInput:          stokInput,
			ipInput:            ipInput,
			passwordInput:      passwordInput,
			logText:            logText,
			singboxConfigInput: singboxConfigInput,
		}
		appWindow.stopSingBox()
	})

	enableSingboxPermanent := widget.NewButton("Enable Sing-box Permanent", func() {
		appWindow := &AppWindow{
			stokInput:     stokInput,
			ipInput:       ipInput,
			passwordInput: passwordInput,
			logText:       logText,
		}
		appWindow.enableSingboxPermanent()
	})

	content := container.NewVBox(
		stokLabel, stokInput,
		ipLabel, ipInput,
		passwordLabel, passwordInput, sshPasswordLabel, sshPasswordInput,
		sendButton, trySSHLoginButton, sshLoginButton, openFileButton, singboxConfigInput, installSingBox, installSingBoxConfig, startSingBox, stopSingBox, enableSingboxPermanent,
		logText,
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

// internal/services/network.go
package services

import (
	"fmt"
	"fyne.io/fyne/v2/widget"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type NetworkService struct{}

func NewNetworkService() *NetworkService {
	return &NetworkService{}
}

func (ns *NetworkService) ScanSubnet(ipInput *widget.Entry) {
	subnet := getLocalSubnet()
	var wg sync.WaitGroup
	resultChan := make(chan string, 1)

	for i := 1; i <= 254; i++ {
		wg.Add(1)
		go checkIP(subnet, i, &wg, resultChan)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	select {
	case ip := <-resultChan:
		ipInput.SetText(ip)
	}
}

func getLocalSubnet() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
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
					return subnet
				}
			}
		}
	}
	return ""
}

func checkIP(subnet string, i int, wg *sync.WaitGroup, resultChan chan<- string) {
	defer wg.Done()
	ip := fmt.Sprintf("%s.%d", subnet, i)

	port := 8099
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

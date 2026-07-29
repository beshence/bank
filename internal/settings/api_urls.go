package settings

import (
	"net"
	"os"
	"slices"
	"strings"
	"sync"
)

var (
	apiUrls  []string
	apiMutex sync.RWMutex
)

func InitAPIUrls(port string) {
	urls := make([]string, 0)

	publishLocalIPs := os.Getenv("BANK_PUBLISH_LOCAL_IPS")
	publishLocalIPsBool := true
	if publishLocalIPs == "false" || publishLocalIPs == "0" {
		publishLocalIPsBool = false
	}

	disableTls := os.Getenv("BANK_DISABLE_TLS")
	disableTlsBool := false
	if disableTls == "true" || disableTls == "1" {
		disableTlsBool = true
	}

	var scheme string
	if disableTlsBool {
		scheme = "http://"
	} else {
		scheme = "https://"
	}

	if publishLocalIPsBool {
		// localhost
		urls = append(urls, scheme+"127.0.0.1:"+port+"/api")
		urls = append(urls, scheme+"localhost:"+port+"/api")

		// local IPs
		ifaces, err := net.Interfaces()
		if err == nil {
			for _, iface := range ifaces {
				addrs, err := iface.Addrs()
				if err != nil {
					continue
				}

				for _, addr := range addrs {
					ip, _, err := net.ParseCIDR(addr.String())
					if err != nil {
						continue
					}

					ipv4 := ip.To4()
					if ipv4 == nil {
						continue
					}

					url := scheme + ipv4.String() + ":" + port + "/api"

					if !contains(urls, url) {
						urls = append(urls, url)
					}
				}
			}
		}

		urls = append(urls, "gateway://gateway.beshence.com")
	}

	// from environment
	envUrls := os.Getenv("BANK_PUBLIC_URLS")

	for url := range strings.SplitSeq(envUrls, ",") {
		url = strings.TrimSpace(url)

		if url != "" && !contains(urls, url) {
			urls = append(urls, url)
		}
	}

	apiMutex.Lock()
	defer apiMutex.Unlock()

	apiUrls = urls
}

func GetAPIUrls() []string {
	apiMutex.RLock()
	defer apiMutex.RUnlock()

	result := make([]string, len(apiUrls))
	copy(result, apiUrls)

	return result
}

func contains(slice []string, value string) bool {
	return slices.Contains(slice, value)
}

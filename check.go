package socks4dialer

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

//You can take single proxy from https://github.com/TheSpeedX/PROXY-List

func doCheckRequest(proxyAddr, dstURL string) (body []byte, err error) {

	proxyURL, err := url.Parse(proxyAddr)
	if err != nil {
		return
	}

	dialer := NewSocks4Dialer(proxyURL)
	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	client := http.Client{Transport: transport, Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", dstURL, nil)

	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	return
}

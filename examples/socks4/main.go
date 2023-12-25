package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/niklak/socks4dialer"
)

func main() {
	// TODO: replace proxyRawURL with yours. You can get a list of proxy here:
	// https://github.com/TheSpeedX/PROXY-List/blob/master/socks4.txt
	proxyRawURL := "socks4://ip:port"
	dstRawURL := "http://icanhazip.com/"

	proxyURL, err := url.Parse(proxyRawURL)
	if err != nil {
		panic(err)
	}

	dialer := socks4dialer.NewSocks4Dialer(proxyURL)
	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	client := http.Client{Transport: transport, Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", dstRawURL, nil)

	if err != nil {
		panic(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Your ip is: %s\n", rawBody)

}

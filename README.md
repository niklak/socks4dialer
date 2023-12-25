# SOCKS4DIALER

This package contains a SOCKS4 protocol dialer to work with a standard (golang) HTTP client.

## Install the package
```
go get -u github.com/niklak/socks4dialer
```

## Import
```
import (
    "github.com/niklak/socks4dialer"
)
```

## Example

```
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
    // Change proxyRawURL with your socks4 proxy address
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

```


## License

Licensed under MIT ([LICENSE](LICENSE) or http://opensource.org/licenses/MIT)
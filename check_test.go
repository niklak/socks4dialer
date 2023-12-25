package socks4dialer

import (
	"reflect"
	"testing"
)

func Test_doCheckRequest(t *testing.T) {
	type args struct {
		proxyAddr string
		dstURL    string
	}
	tests := []struct {
		name     string
		args     args
		wantBody []byte
		wantErr  bool
	}{
		// TODO: replaces proxy addresses with yours.
		{name: "test socks4 http request #0",
			args:     args{proxyAddr: "socks4://ip:port", dstURL: "http://icanhazip.com"},
			wantBody: []byte("47.243.50.83:80"),
			wantErr:  false,
		},

		{name: "test socks4 https request #0",
			args:     args{proxyAddr: "socks4://ip:port", dstURL: "http://icanhazip.com"},
			wantBody: []byte("123.182.58.227:8089"),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBody, err := doCheckRequest(tt.args.proxyAddr, tt.args.dstURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("doCheckRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotBody, tt.wantBody) {
				t.Errorf("doCheckRequest() = %s, want %s", gotBody, tt.wantBody)
			}
		})
	}
}

// Package socks4dialer provides a SOCKS version 4 client implementation.

package socks4dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	socksnoDeadline   = time.Time{}
	socksaLongTimeAgo = time.Unix(1, 0)
)

const (
	schemeSocks4  = "socks4"
	schemeSocks4a = "socks4a"
	socksVersion4 = 0x04
)

// A Command represents a SOCKS command.
type socksCommand int

const (
	socksCmdConnect socksCommand = 0x01 // establishes an active-open forward proxy connection
	sockscmdBind    socksCommand = 0x02 // establishes a passive-open forward proxy connection

)

func (cmd socksCommand) String() string {
	switch cmd {
	case socksCmdConnect:
		return "socks connect"
	case sockscmdBind:
		return "socks bind"
	default:
		return "socks " + strconv.Itoa(int(cmd))
	}
}

// Wire protocol constants.
type socksReply int

const socksStatusSucceeded socksReply = 90

// A Reply represents a SOCKS command reply code.

func (code socksReply) String() string {
	switch code {
	case socksStatusSucceeded:
		return "request granted"
	case 91:
		return "request rejected or failed"
	case 92:
		return "request rejected because SOCKS server cannot connect to identd on the client"
	case 93:
		return "request rejected because the client program and identd report different user-ids"
	default:
		return "unknown code: " + strconv.Itoa(int(code))
	}
}

// An Addr represents a SOCKS-specific address.
// Either Name or IP is used exclusively.
type socksAddr struct {
	Name string // fully-qualified domain name
	IP   net.IP
	Port int
}

func (a *socksAddr) Network() string { return "socks" }

func (a *socksAddr) String() string {
	if a == nil {
		return "<nil>"
	}
	port := strconv.Itoa(a.Port)
	if a.IP == nil {
		return net.JoinHostPort(a.Name, port)
	}
	return net.JoinHostPort(a.IP.String(), port)
}

// A Conn represents a forward proxy connection.
type socksConn struct {
	net.Conn
	boundAddr net.Addr
}

// BoundAddr returns the address assigned by the proxy server for
// connecting to the command target address from the proxy server.
func (c *socksConn) BoundAddr() net.Addr {
	if c == nil {
		return nil
	}
	return c.boundAddr
}

// A Socks4Dialer holds SOCKS4-specific options.
type Socks4Dialer struct {
	cmd          socksCommand // either CmdConnect or cmdBind
	proxyNetwork string       // network between a proxy server and a client
	proxyAddress string       // proxy server address
	scheme       string       // socks -- socks4 or socks4a
	// ProxyDial specifies the optional dial function for
	// establishing the transport connection.
	ProxyDial func(context.Context, string, string) (net.Conn, error)
}

func (d *Socks4Dialer) connect(ctx context.Context, c net.Conn, address string) (_ net.Addr, ctxErr error) {
	host, port, err := socksSplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok && !deadline.IsZero() {
		c.SetDeadline(deadline)
		defer c.SetDeadline(socksnoDeadline)
	}
	if ctx != context.Background() {
		errCh := make(chan error, 1)
		done := make(chan struct{})
		defer func() {
			close(done)
			if ctxErr == nil {
				ctxErr = <-errCh
			}
		}()
		go func() {
			select {
			case <-ctx.Done():
				c.SetDeadline(socksaLongTimeAgo)
				errCh <- ctx.Err()
			case <-done:
				errCh <- nil
			}
		}()
	}

	ip := net.IPv4(0, 0, 0, 1).To4()

	if d.scheme == schemeSocks4 {
		if ip, err = lookupIPV4(host); err != nil {
			return
		}
	}

	req := []byte{
		socksVersion4,
		byte(d.cmd),
		byte(port >> 8),
		byte(port),
	}
	req = append(req, ip...) // special invalid IP address to indicate the host name is provided
	//no userid
	req = append(req, 0) //`null string`

	if d.scheme == schemeSocks4a {
		req = append(req, []byte(host)...)
		req = append(req, byte('\x00'))
	}

	if _, ctxErr = c.Write(req); ctxErr != nil {
		return
	}

	resp := make([]byte, 8)

	if _, ctxErr = io.ReadFull(c, resp); ctxErr != nil {
		return
	}

	if cmdErr := socksReply(resp[1]); cmdErr != socksStatusSucceeded {
		return nil, errors.New(cmdErr.String())
	}

	var a socksAddr
	a.IP = make(net.IP, net.IPv4len)
	copy(a.IP, resp[3:])
	a.Port = int(resp[2])<<8 | int(resp[3])
	return &a, nil
}

// DialContext connects to the provided address on the provided
// network.
//
// The returned error value may be a net.OpError. When the Op field of
// net.OpError contains "socks", the Source field contains a proxy
// server address and the Addr field contains a command target
// address.
//
// See func Dial of the net package of standard library for a
// description of the network and address parameters.
func (d *Socks4Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if err := d.validateTarget(network, address); err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: d.cmd.String(), Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if ctx == nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: d.cmd.String(), Net: network, Source: proxy, Addr: dst, Err: errors.New("nil context")}
	}
	var err error
	var c net.Conn
	if d.ProxyDial != nil {
		c, err = d.ProxyDial(ctx, d.proxyNetwork, d.proxyAddress)
	} else {
		var dd net.Dialer
		c, err = dd.DialContext(ctx, d.proxyNetwork, d.proxyAddress)
	}

	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: d.cmd.String(), Net: network, Source: proxy, Addr: dst, Err: err}
	}

	a, err := d.connect(ctx, c, address)
	if err != nil {
		c.Close()
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: d.cmd.String(), Net: network, Source: proxy, Addr: dst, Err: err}
	}

	return &socksConn{Conn: c, boundAddr: a}, nil
}

// DialWithConn initiates a connection from SOCKS server to the target
// network and address using the connection c that is already
// connected to the SOCKS server.
//
// It returns the connection's local address assigned by the SOCKS
// server.
func (d *Socks4Dialer) DialWithConn(ctx context.Context, c net.Conn, network, address string) (net.Addr, error) {
	if err := d.validateTarget(network, address); err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: d.cmd.String(), Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if ctx == nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: d.cmd.String(), Net: network, Source: proxy, Addr: dst, Err: errors.New("nil context")}
	}
	a, err := d.connect(ctx, c, address)
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: d.cmd.String(), Net: network, Source: proxy, Addr: dst, Err: err}
	}
	return a, nil
}

// Dial connects to the provided address on the provided network.
//
// Unlike DialContext, it returns a raw transport connection instead
// of a forward proxy connection.
//
// Deprecated: Use DialContext or DialWithConn instead.
func (d *Socks4Dialer) Dial(network, address string) (net.Conn, error) {
	if err := d.validateTarget(network, address); err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: d.cmd.String(), Net: network, Source: proxy, Addr: dst, Err: err}
	}
	var err error
	var c net.Conn
	if d.ProxyDial != nil {
		c, err = d.ProxyDial(context.Background(), d.proxyNetwork, d.proxyAddress)
	} else {
		c, err = net.Dial(d.proxyNetwork, d.proxyAddress)
	}
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: d.cmd.String(), Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if _, err := d.DialWithConn(context.Background(), c, network, address); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func (d *Socks4Dialer) validateTarget(network, address string) error {
	switch network {
	case "tcp", "tcp6", "tcp4":
	default:
		return errors.New("network not implemented")
	}
	switch d.cmd {
	case socksCmdConnect, sockscmdBind:
	default:
		return errors.New("command not implemented")
	}
	return nil
}

func (d *Socks4Dialer) pathAddrs(address string) (proxy, dst net.Addr, err error) {
	for i, s := range []string{d.proxyAddress, address} {
		host, port, err := socksSplitHostPort(s)
		if err != nil {
			return nil, nil, err
		}
		a := &socksAddr{Port: port}
		a.IP = net.ParseIP(host)
		if a.IP == nil {
			a.Name = host
		}
		if i == 0 {
			proxy = a
		} else {
			dst = a
		}
	}
	return
}

func NewSocks4Dialer(proxyURL *url.URL) (d *Socks4Dialer) {
	pad := strings.Builder{}
	pad.WriteString(proxyURL.Hostname())
	pad.WriteString(":")
	pad.WriteString(proxyURL.Port())
	d = &Socks4Dialer{
		proxyNetwork: "tcp",
		proxyAddress: pad.String(),
		scheme:       proxyURL.Scheme,
		cmd:          socksCmdConnect,
	}

	return
}

package utp

import (
	"bufio"
	"crypto/tls"
	"encoding/gob"
	"net"

	"github.com/anacrolix/utp"
	"github.com/micro/go-micro/transport"
	mls "github.com/micro/misc/lib/tls"
)

func (u *utpTransport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	dopts := transport.DialOptions{
		Timeout: transport.DefaultDialTimeout,
	}

	for _, opt := range opts {
		opt(&dopts)
	}

	c, err := utp.DialTimeout(addr, dopts.Timeout)
	if err != nil {
		return nil, err
	}

	if u.opts.Secure || u.opts.TLSConfig != nil {
		config := u.opts.TLSConfig
		if config == nil {
			config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		c = tls.Client(c, config)
	}

	encBuf := bufio.NewWriter(c)

	return &utpClient{
		dialOpts: dopts,
		conn:     c,
		encBuf:   encBuf,
		enc:      gob.NewEncoder(encBuf),
		dec:      gob.NewDecoder(c),
		timeout:  u.opts.Timeout,
	}, nil
}

func (u *utpTransport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	var options transport.ListenOptions
	for _, o := range opts {
		o(&options)
	}

	var l net.Listener
	var err error

	if u.opts.Secure || u.opts.TLSConfig != nil {
		config := u.opts.TLSConfig

		fn := func(addr string) (net.Listener, error) {
			if config == nil {
				hosts := []string{addr}

				// check if its a valid host:port
				if host, _, err := net.SplitHostPort(addr); err == nil {
					if len(host) == 0 {
						hosts = getIPAddrs()
					} else {
						hosts = []string{host}
					}
				}

				// generate a certificate
				cert, err := mls.Certificate(hosts...)
				if err != nil {
					return nil, err
				}
				config = &tls.Config{Certificates: []tls.Certificate{cert}}
			}
			l, err := utp.Listen(addr)
			if err != nil {
				return nil, err
			}
			return tls.NewListener(l, config), nil
		}

		l, err = listen(addr, fn)
	} else {
		l, err = listen(addr, utp.Listen)
	}

	if err != nil {
		return nil, err
	}

	return &utpListener{
		t:    u.opts.Timeout,
		l:    l,
		opts: options,
	}, nil
}

func (u *utpTransport) String() string {
	return "utp"
}

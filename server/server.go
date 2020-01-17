// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
)

// ServerOpts contains parameters for server.NewServer()
type ServerOpts struct {
	// The factory that will be used to create a new FTPDriver instance for
	// each client connection. This is a mandatory option.
	Factory DriverFactory

	Auth Auth

	// Server Name, Default is Go Ftp Server
	Name string

	// The hostname that the FTP server should listen on. Optional, defaults to
	// "::", which means all hostnames on ipv4 and ipv6.
	Hostname string

	// Public IP of the server
	PublicIp string

	// Passive ports
	PassivePorts string

	// The port that the FTP should listen on. Optional, defaults to 3000. In
	// a production environment you will probably want to change this to 21.
	Port int

	// use tls, default is false
	TLS bool

	// if tls used, cert file is required
	CertFile string

	// if tls used, key file is required
	KeyFile string

	// If ture TLS is used in RFC4217 mode
	ExplicitFTPS bool

	WelcomeMessage string

	// A logger implementation, if nil the StdLogger is used
	Logger Logger
}

// Server is the root of your FTP application. You should instantiate one
// of these and call ListenAndServe() to start accepting client connections.
//
// Always use the NewServer() method to create a new Server.
type Server struct {
	*ServerOpts
	listenTo  string
	logger    Logger
	listener  net.Listener
	tlsConfig *tls.Config
	ctx       context.Context
	cancel    context.CancelFunc
	feats     string
}

// ErrServerClosed is returned by ListenAndServe() or Serve() when a shutdown
// was requested.
var ErrServerClosed = errors.New("ftp: Server closed")

// serverOptsWithDefaults copies an ServerOpts struct into a new struct,
// then adds any default values that are missing and returns the new data.
func serverOptsWithDefaults(opts *ServerOpts) *ServerOpts {
	var newOpts ServerOpts
	if opts == nil {
		opts = &ServerOpts{}
	}

	if opts.Hostname == "" {
		newOpts.Hostname = "::"
	} else {
		newOpts.Hostname = opts.Hostname
	}

	if opts.Port == 0 {
		newOpts.Port = 3000
	} else {
		newOpts.Port = opts.Port
	}

	newOpts.Factory = opts.Factory
	if opts.Name == "" {
		newOpts.Name = "Go FTP Server"
	} else {
		newOpts.Name = opts.Name
	}

	if opts.WelcomeMessage == "" {
		newOpts.WelcomeMessage = defaultWelcomeMessage
	} else {
		newOpts.WelcomeMessage = opts.WelcomeMessage
	}

	if opts.Auth != nil {
		newOpts.Auth = opts.Auth
	}

	newOpts.Logger = &StdLogger{}
	if opts.Logger != nil {
		newOpts.Logger = opts.Logger
	}

	newOpts.TLS = opts.TLS
	newOpts.KeyFile = opts.KeyFile
	newOpts.CertFile = opts.CertFile
	newOpts.ExplicitFTPS = opts.ExplicitFTPS

	newOpts.PublicIp = opts.PublicIp
	newOpts.PassivePorts = opts.PassivePorts

	return &newOpts
}

// NewServer initialises a new FTP server. Configuration options are provided
// via an instance of ServerOpts. Calling this function in your code will
// probably look something like this:
//
//     factory := &MyDriverFactory{}
//     server  := server.NewServer(&server.ServerOpts{ Factory: factory })
//
// or:
//
//     factory := &MyDriverFactory{}
//     opts    := &server.ServerOpts{
//       Factory: factory,
//       Port: 2000,
//       Hostname: "127.0.0.1",
//     }
//     server  := server.NewServer(opts)
//
func NewServer(opts *ServerOpts) *Server {
	opts = serverOptsWithDefaults(opts)
	s := new(Server)
	s.ServerOpts = opts
	s.listenTo = net.JoinHostPort(opts.Hostname, strconv.Itoa(opts.Port))
	s.logger = opts.Logger
	return s
}

// NewConn constructs a new object that will handle the FTP protocol over
// an active net.TCPConn. The TCP connection should already be open before
// it is handed to this functions. driver is an instance of FTPDriver that
// will handle all auth and persistence details.
func (s *Server) newConn(tcpConn net.Conn, driver Driver) *Conn {
	conn := new(Conn)
	conn.namePrefix = "/"
	conn.conn = tcpConn
	conn.controlReader = bufio.NewReader(tcpConn)
	conn.controlWriter = bufio.NewWriter(tcpConn)
	conn.driver = driver
	conn.auth = s.Auth
	conn.server = s
	conn.sessionID = newSessionID()
	conn.logger = s.logger
	conn.tlsConfig = s.tlsConfig

	driver.Init(conn)
	return conn
}

func (s *Server)doLogger(sessionId string, format string, v ...interface{})  {
	s.logger.Printf(sessionId, format, v ...)
}

func simpleTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	config := &tls.Config{}
	if config.NextProtos == nil {
		config.NextProtos = []string{"ftp"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// ListenAndServe asks a new Server to begin accepting client connections. It
// accepts no arguments - all configuration is provided via the NewServer
// function.
//
// If the server fails to start for any reason, an error will be returned. Common
// errors are trying to bind to a privileged port or something else is already
// listening on the same port.
//
func (s *Server) ListenAndServe() error {
	var listener net.Listener
	var err error
	var curFeats = featCmds

	if s.ServerOpts.TLS {
		s.tlsConfig, err = simpleTLSConfig(s.CertFile, s.KeyFile)
		if err != nil {
			return err
		}

		curFeats += " AUTH TLS\n PBSZ\n PROT\n"

		if s.ServerOpts.ExplicitFTPS {
			listener, err = net.Listen("tcp", s.listenTo)
		} else {
			listener, err = tls.Listen("tcp", s.listenTo, s.tlsConfig)
		}
	} else {
		listener, err = net.Listen("tcp", s.listenTo)
	}

	if err != nil {
		return err
	}
	s.feats = fmt.Sprintf(feats, curFeats)

	s.doLogger("", "%s listening on %d", s.Name, s.Port)

	return s.Serve(listener)
}

// Serve accepts connections on a given net.Listener and handles each
// request in a new goroutine.
//
func (s *Server) Serve(l net.Listener) error {
	s.listener = l
	s.ctx, s.cancel = context.WithCancel(context.Background())
	sessionID := ""
	for {
		tcpConn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return ErrServerClosed
			default:
			}
			s.logger.Printf(sessionID, "listening error: %v", err)
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return err
		}
		driver, err := s.Factory.NewDriver()
		if err != nil {
			s.logger.Printf(sessionID, "Error creating driver, aborting client connection: %v", err)
			tcpConn.Close()
		} else {
			ftpConn := s.newConn(tcpConn, driver)
			go ftpConn.Serve()
		}
	}
}

// Shutdown will gracefully stop a server. Already connected clients will retain their connections
func (s *Server) Shutdown() error {
	if s.cancel != nil {
		s.cancel()
	}

	if s.listener != nil {
		return s.listener.Close()
	}

	// s wasnt even started
	return nil
}

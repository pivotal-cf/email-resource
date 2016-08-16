package email_resource_test

import (
	"net"

	"bitbucket.org/chrj/smtpd"
)

type FakeSMTPServer struct {
	listener   net.Listener
	server     *smtpd.Server
	Deliveries []smtpd.Envelope
	Host       string
	Port       string
}

func NewFakeSMTPServer() *FakeSMTPServer {
	return &FakeSMTPServer{
		server: &smtpd.Server{
			Hostname: "127.0.0.1:0",
		},
		Deliveries: make([]smtpd.Envelope, 0),
	}
}

func (s *FakeSMTPServer) Boot() {
	var err error
	s.listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	s.server.Handler = func(peer smtpd.Peer, env smtpd.Envelope) error {
		s.Deliveries = append(s.Deliveries, env)
		return nil
	}

	go s.server.Serve(s.listener)

	addr := s.listener.Addr().String()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		panic(err)
	}
	s.Host = host
	s.Port = port
}

func (s *FakeSMTPServer) Close() {
	s.listener.Close()
}

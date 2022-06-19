package server

import (
	"context"
	"serverless-lb/pkg/resolver"
	"time"

	"github.com/miekg/dns"
)

type Server struct {
	Server   *dns.Server
	Resolver *resolver.Resolver
}

func (s *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		msg.Answer = s.Resolver.Answer(msg.Question...)
	}

	_ = w.WriteMsg(msg)
}

func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		ctxShutDown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer func() {
			cancel()
		}()

		_ = s.Server.ShutdownContext(ctxShutDown)
	}()

	s.Server.Handler = s

	return s.Server.ListenAndServe()
}

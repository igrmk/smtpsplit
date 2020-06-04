package main

import (
	"io"
	"net"
	"net/smtp"
	"time"

	"github.com/igrmk/go-smtpd/smtpd"
)

var notResponding = smtpd.SMTPError("441 Server is not responding")

type sess struct {
	domain string
	client *smtp.Client
	data   io.WriteCloser
}

type env struct {
	*smtpd.BasicEnvelope
	from              smtpd.MailAddress
	domainToSess      map[string]*sess
	domainToRecipient map[string][]string
	routes            map[string]string
	size              *int
	timeout           time.Duration
	host              string
}

// AddRecipient implements smtpd.Envelope.AddRecipient
func (e *env) AddRecipient(rcpt smtpd.MailAddress) error {
	if e.domainToRecipient == nil {
		e.domainToRecipient = make(map[string][]string)
	}
	domain := rcpt.Hostname()
	if _, ok := e.routes[domain]; ok {
		e.domainToRecipient[domain] = append(e.domainToRecipient[domain], rcpt.Email())
	}
	return e.BasicEnvelope.AddRecipient(rcpt)
}

// BeginData implements smtpd.Envelope.BeginData
func (e *env) BeginData() error {
	if err := e.initSessions(); err != nil {
		return err
	}
	if err := e.forAllSessions(sendHello); err != nil {
		return err
	}
	if err := e.forAllSessions(sendMail); err != nil {
		return err
	}
	if err := e.forAllSessions(sendRcpts); err != nil {
		return err
	}
	if err := e.forAllSessions(beginData); err != nil {
		return err
	}
	return nil
}

// Write implements smtpd.Envelope.Write
func (e *env) Write(line []byte) error {
	appendData := func(_ *env, s *sess) error {
		if _, err := s.data.Write(line); err != nil {
			lerr("could not send DATA, %v", err)
			return notResponding
		}
		return nil
	}
	if err := e.forAllSessions(appendData); err != nil {
		return err
	}
	return nil
}

// Close implements smtpd.Envelope.Close
func (e *env) Close() error {
	if err := e.forAllSessions(closeConnection); err != nil {
		return err
	}
	return nil
}

func (e *env) forAllSessions(f func(*env, *sess) error) error {
	for _, sess := range e.domainToSess {
		if err := f(e, sess); err != nil {
			return err
		}
	}
	return nil
}

func (e *env) initSessions() error {
	addresses := make(map[string]string)
	for domain := range e.domainToRecipient {
		server := e.routes[domain]
		if server != "" {
			addresses[server] = domain
		}
	}
	sessions := make(map[string]*sess)
	for s, domain := range addresses {
		conn, err := net.DialTimeout("tcp", s, e.timeout)
		if err != nil {
			lerr("could not dial, %v", err)
			return notResponding
		}
		if e.timeout != 0 {
			deadline := time.Now().Add(e.timeout)
			checkErr(conn.SetReadDeadline(deadline))
			checkErr(conn.SetWriteDeadline(deadline))
		}
		client, err := smtp.NewClient(conn, domain)
		if err != nil {
			lerr("server is not responding", err)
			return notResponding
		}
		sessions[domain] = &sess{client: client, domain: domain}
	}
	e.domainToSess = sessions
	return nil
}

func sendHello(e *env, s *sess) error {
	if err := s.client.Hello(e.host); err != nil {
		lerr("could not send HELO, %v", err)
		return notResponding
	}
	return nil
}

func sendMail(e *env, s *sess) error {
	if err := s.client.Mail(e.from.Email()); err != nil {
		lerr("could not send MAIL, %v", err)
		return notResponding
	}
	ldbg("MAIL OK")
	return nil
}

func sendRcpts(e *env, s *sess) error {
	for _, r := range e.domainToRecipient[s.domain] {
		if err := s.client.Rcpt(r); err != nil {
			lerr("could not send RCPT, %v", err)
			return notResponding
		}
	}
	ldbg("RCPT OK")
	return nil
}

func beginData(_ *env, s *sess) error {
	cw, err := s.client.Data()
	if err != nil {
		lerr("could not send DATA, %v", err)
		return notResponding
	}
	s.data = cw
	ldbg("DATA OK")
	return nil
}

func closeConnection(_ *env, s *sess) error {
	if err := s.data.Close(); err != nil {
		lerr("could not flush data, %v", err)
		return notResponding
	}
	if err := s.client.Quit(); err != nil {
		lerr("could not send QUIT, %v", err)
		return notResponding
	}
	return nil
}

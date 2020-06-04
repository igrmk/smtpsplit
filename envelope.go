package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/igrmk/go-smtpd/smtpd"
)

type sess struct {
	rwc    net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

type env struct {
	*smtpd.BasicEnvelope
	from              smtpd.MailAddress
	domainToSess      map[string]sess
	domainToRecipient map[string][]string
	routes            map[string]string
	size              *int
	timeout           time.Duration
}

// Close implements smtpd.Envelope.Close
func (e *env) Close() error {
	if err := e.sendToAll(".", "250"); err != nil {
		return err
	}
	if err := e.sendToAll("QUIT", "221"); err != nil {
		return err
	}
	for _, sess := range e.domainToSess {
		if err := sess.writer.Flush(); err != nil {
			lerr("could not flush data, %v", err)
			return smtpd.SMTPError("441 Server is not responding")
		}
		if err := sess.rwc.Close(); err != nil {
			lerr("could not close connection, %v", err)
			return smtpd.SMTPError("441 Server is not responding")
		}
	}
	return nil
}

// BeginData implements smtpd.Envelope.BeginData
func (e *env) BeginData() error {
	servers := make(map[string]string)
	for d := range e.domainToRecipient {
		server := e.routes[d]
		if server != "" {
			servers[server] = d
		}
	}
	sessions := make(map[string]sess)
	for s, h := range servers {
		conn, err := net.DialTimeout("tcp", s, e.timeout)
		if err != nil {
			lerr("could not dial, %v", err)
			return smtpd.SMTPError("441 Server is not responding")
		}
		if e.timeout != 0 {
			deadline := time.Now().Add(e.timeout)
			checkErr(conn.SetReadDeadline(deadline))
			checkErr(conn.SetWriteDeadline(deadline))
		}
		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)
		sessions[h] = sess{rwc: conn, reader: r, writer: w}
	}
	e.domainToSess = sessions
	for _, sess := range e.domainToSess {
		if err := checkOK(sess.reader, "220"); err != nil {
			return err
		}
	}
	if err := e.sendToAll("HELO", "250"); err != nil {
		return err
	}
	if err := e.sendFrom(); err != nil {
		return err
	}
	if err := e.sendRecepients(); err != nil {
		return err
	}
	if err := e.sendToAll("DATA", "354"); err != nil {
		return err
	}
	return nil
}

// Write implements smtpd.Envelope.Write
func (e *env) Write(line []byte) error {
	for _, sess := range e.domainToSess {
		ldbg("sending data line to all: %q", line)
		if _, err := sess.writer.Write(line); err != nil {
			lerr("could not write, %v", err)
			return smtpd.SMTPError("441 Server is not responding")
		}
	}
	return nil
}

func (e *env) sendToAll(text string, code string) error {
	text += "\r\n"
	ldbg("send to all: %q", text)
	for _, sess := range e.domainToSess {
		if _, err := sess.writer.Write([]byte(text)); err != nil {
			lerr("could not write, %v", err)
			return smtpd.SMTPError("441 Server is not responding")
		}
		if err := sess.writer.Flush(); err != nil {
			lerr("could not flush data, %v", err)
			return smtpd.SMTPError("441 Server is not responding")
		}
		if err := checkOK(sess.reader, code); err != nil {
			return err
		}
	}
	return nil
}

func checkOK(r *bufio.Reader, code string) error {
	for {
		s, err := r.ReadString('\n')
		if err != nil {
			lerr("could not read data, %v", err)
			return smtpd.SMTPError("441 Server is not responding")
		}
		ldbg("got: %q", s)
		if strings.HasPrefix(s, code+"-") {
			continue
		}
		if !strings.HasPrefix(s, code) {
			lerr("server returned error %q", s)
			return smtpd.SMTPError(s)
		}
		return nil
	}
}

func (e *env) sendFrom() error {
	var text string
	if e.size != nil {
		text = fmt.Sprintf("MAIL FROM:<%s> %d", e.from, *e.size)
	} else {
		text = fmt.Sprintf("MAIL FROM:<%s>", e.from)
	}
	if err := e.sendToAll(text, "250"); err != nil {
		return err
	}
	return nil
}

func (e *env) sendRecepients() error {
	if e.domainToRecipient == nil {
		return nil
	}
	for d, rs := range e.domainToRecipient {
		sess := e.domainToSess[d]
		for _, r := range rs {
			text := fmt.Sprintf("RCPT TO:<%s>\r\n", r)
			ldbg("sending recipient: %q", text)
			if _, err := sess.writer.Write([]byte(text)); err != nil {
				lerr("could not write data, %v", err)
				return smtpd.SMTPError("441 Server is not responding")
			}
			if err := sess.writer.Flush(); err != nil {
				lerr("could not flush data, %v", err)
				return smtpd.SMTPError("441 Server is not responding")
			}
			if err := checkOK(sess.reader, "250"); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddRecipient implements smtpd.Envelope.AddRecipient
func (e *env) AddRecipient(rcpt smtpd.MailAddress) error {
	if e.domainToRecipient == nil {
		e.domainToRecipient = make(map[string][]string)
	}
	_, domain := splitAddress(rcpt.Email())
	if _, ok := e.routes[domain]; ok {
		e.domainToRecipient[domain] = append(e.domainToRecipient[domain], rcpt.Email())
	}
	return e.BasicEnvelope.AddRecipient(rcpt)
}

func splitAddress(a string) (string, string) {
	a = strings.ToLower(a)
	parts := strings.Split(a, "@")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

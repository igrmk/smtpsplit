package main

import (
	"crypto/tls"
	"encoding/json"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/igrmk/go-smtpd/smtpd"
)

// checkErr panics on an error
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

type worker struct {
	cfg     *config
	tls     *tls.Config
	timeout time.Duration
}

func newWorker() *worker {
	if len(os.Args) != 2 {
		panic("usage: smtpsplit <config>")
	}
	cfg := readConfig(os.Args[1])
	timeout := time.Second * time.Duration(cfg.TimeoutSeconds)
	w := &worker{
		cfg:     cfg,
		timeout: timeout,
	}
	if cfg.Certificate != "" {
		tls, err := loadTLS(cfg.Certificate, cfg.CertificateKey)
		checkErr(err)
		w.tls = tls
	}

	return w
}

func envelopeFactory(
	routes map[string]string,
	timeout time.Duration,
	host string,
	tls *tls.Config,
	debug bool,
) func(
	smtpd.Connection,
	smtpd.MailAddress,
	*int,
) (
	smtpd.Envelope,
	error,
) {
	return func(c smtpd.Connection, from smtpd.MailAddress, size *int) (smtpd.Envelope, error) {
		return &env{
			BasicEnvelope: &smtpd.BasicEnvelope{},
			from:          from,
			size:          size,
			routes:        routes,
			timeout:       timeout,
			host:          host,
			tls:           tls,
			debug:         debug,
		}, nil
	}
}

func (w *worker) logConfig() {
	cfgString, err := json.MarshalIndent(w.cfg, "", "    ")
	checkErr(err)
	linf("config: " + string(cfgString))
}

func loadTLS(certFile string, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())
	w := newWorker()
	w.logConfig()

	smtp := &smtpd.Server{
		Hostname:     w.cfg.Host,
		Addr:         w.cfg.ListenAddress,
		OnNewMail:    envelopeFactory(w.cfg.Routes, w.timeout, w.cfg.Host, w.tls, w.cfg.Debug),
		TLSConfig:    w.tls,
		Log:          lsmtpd,
		ReadTimeout:  w.timeout,
		WriteTimeout: w.timeout,
	}
	go func() {
		err := smtp.ListenAndServe()
		checkErr(err)
	}()
	signals := make(chan os.Signal, 16)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	s := <-signals
	linf("got signal %v", s)
}

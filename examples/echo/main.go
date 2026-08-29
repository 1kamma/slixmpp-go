// Command echo connects an XMPP client, sends initial presence, and echoes
// direct chat messages. It is intentionally small enough to use as a transport
// smoke test.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/1kamma/slixmpp-go/xep"
	"github.com/1kamma/slixmpp-go/xmpp"
)

func main() {
	var (
		jid       = flag.String("jid", os.Getenv("XMPP_JID"), "account JID")
		password  = flag.String("password", os.Getenv("XMPP_PASSWORD"), "account password (prefer XMPP_PASSWORD)")
		address   = flag.String("address", "", "optional host:port override")
		resource  = flag.String("resource", "slixmpp-go-echo", "resource to bind")
		directTLS = flag.Bool("direct-tls", false, "use direct TLS instead of STARTTLS")
		debug     = flag.Bool("debug", false, "enable connection diagnostics")
		debugXML  = flag.Bool("debug-xml", false, "log stanza XML; SASL payloads are redacted")
	)
	flag.Parse()

	// For compatibility with simple command lines, two trailing arguments may
	// supply jid and password: go run ./examples/echo --debug JID PASSWORD.
	if *jid == "" && flag.NArg() > 0 {
		*jid = flag.Arg(0)
	}
	if *password == "" && flag.NArg() > 1 {
		*password = flag.Arg(1)
	}
	if *jid == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "usage: echo [flags] JID PASSWORD")
		flag.PrintDefaults()
		os.Exit(2)
	}

	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
	config := xmpp.DefaultConfig(*jid, *password)
	config.Resource = *resource
	config.Address = *address
	config.DirectTLS = *directTLS
	config.Debug = *debug
	config.DebugXML = *debugXML
	config.Logger = logger

	client, err := xmpp.NewClient(config)
	if err != nil {
		logger.Fatal(err)
	}
	if err := xep.RegisterAll(client); err != nil {
		logger.Fatal(err)
	}
	if err := xep.LoadDefaults(client); err != nil {
		logger.Fatal(err)
	}

	client.OnError = func(err error) { logger.Printf("XMPP error: %v", err) }
	client.OnMessage = func(message xmpp.Message) {
		if message.Type == xmpp.MessageGroupChat || message.Type == xmpp.MessageError || message.Body == "" {
			return
		}
		logger.Printf("message from=%s body=%q", message.From, message.Body)
		reply := message.Reply(message.Body)
		reply.ID = client.NextID()
		if err := client.Send(reply); err != nil {
			logger.Printf("echo failed: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := client.Connect(ctx); err != nil {
		logger.Fatal(err)
	}
	logger.Printf("connected as %s", client.BoundJID())

	select {
	case <-ctx.Done():
		_ = client.Close()
	case <-client.Done():
	}
}

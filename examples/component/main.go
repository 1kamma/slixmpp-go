// Command component runs a minimal XEP-0114 echo component.
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
	jid := flag.String("jid", "", "component domain, for example weather.example.org")
	secret := flag.String("secret", os.Getenv("XMPP_COMPONENT_SECRET"), "component secret")
	address := flag.String("address", "", "server component endpoint, for example example.org:5347")
	directTLS := flag.Bool("direct-tls", false, "wrap the component connection in direct TLS")
	debug := flag.Bool("debug", false, "enable diagnostics")
	flag.Parse()
	if *jid == "" || *secret == "" || *address == "" {
		fmt.Fprintln(os.Stderr, "usage: component -jid DOMAIN -address HOST:PORT -secret SECRET")
		flag.PrintDefaults()
		os.Exit(2)
	}

	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
	config := xmpp.DefaultConfig(*jid, "")
	config.ComponentSecret = *secret
	config.Address = *address
	config.DirectTLS = *directTLS
	config.Debug = *debug
	config.Logger = logger
	client, err := xmpp.NewClient(config)
	if err != nil {
		logger.Fatal(err)
	}
	if err := xep.RegisterAll(client); err != nil {
		logger.Fatal(err)
	}
	client.OnMessage = func(message xmpp.Message) {
		if message.Body == "" || message.Type == xmpp.MessageError {
			return
		}
		reply := message.Reply("component echo: " + message.Body)
		reply.ID = client.NextID()
		reply.From = message.To
		if err := client.Send(reply); err != nil {
			logger.Printf("reply failed: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := client.Connect(ctx); err != nil {
		logger.Fatal(err)
	}
	logger.Printf("component authenticated as %s", client.JID())
	select {
	case <-ctx.Done():
		_ = client.Close()
	case <-client.Done():
	}
}

// Command muc joins a room, prints groupchat messages, and sends lines read
// from standard input.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/saret/slixmpp-go/xep"
	"github.com/saret/slixmpp-go/xmpp"
)

func main() {
	jid := flag.String("jid", os.Getenv("XMPP_JID"), "account JID")
	password := flag.String("password", os.Getenv("XMPP_PASSWORD"), "account password")
	room := flag.String("room", "", "room JID")
	nick := flag.String("nick", "go-bot", "room nickname")
	roomPassword := flag.String("room-password", "", "optional room password")
	address := flag.String("address", "", "optional host:port override")
	debug := flag.Bool("debug", false, "enable connection diagnostics")
	flag.Parse()
	if *jid == "" || *password == "" || *room == "" {
		fmt.Fprintln(os.Stderr, "usage: muc -jid JID -password PASSWORD -room ROOM [flags]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
	config := xmpp.DefaultConfig(*jid, *password)
	config.Address = *address
	config.Resource = "slixmpp-go-muc"
	config.Debug = *debug
	config.Logger = logger
	client, err := xmpp.NewClient(config)
	if err != nil {
		logger.Fatal(err)
	}
	if err := xep.RegisterAll(client); err != nil {
		logger.Fatal(err)
	}
	plugin, err := xep.Load(client, 45)
	if err != nil {
		logger.Fatal(err)
	}
	muc, ok := plugin.(*xep.MUC)
	if !ok {
		logger.Fatalf("xep_0045 has unexpected type %T", plugin)
	}

	client.OnError = func(err error) { logger.Printf("XMPP error: %v", err) }
	client.OnMessage = func(message xmpp.Message) {
		if message.Type == xmpp.MessageGroupChat && message.Body != "" {
			logger.Printf("%s: %s", message.From, message.Body)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := client.Connect(ctx); err != nil {
		logger.Fatal(err)
	}
	joined, err := muc.Join(ctx, *room, *nick, xep.JoinOptions{Password: *roomPassword})
	if err != nil {
		_ = client.Close()
		logger.Fatal(err)
	}
	logger.Printf("joined %s as %s", joined.JID, joined.Nick)

	lines := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	for {
		select {
		case line, open := <-lines:
			if !open {
				_ = muc.Leave(*room, "stdin closed")
				_ = client.Close()
				return
			}
			if err := muc.SendMessage(*room, line); err != nil {
				logger.Printf("send failed: %v", err)
			}
		case <-ctx.Done():
			_ = muc.Leave(*room, "client exiting")
			_ = client.Close()
			return
		case <-client.Done():
			return
		}
	}
}

package xmpp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"math/big"
	"net"
	"testing"
	"time"
)

func TestClientSTARTTLSPlainBindPresenceAndMessage(t *testing.T) {
	certificate := testCertificate(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverErrors := make(chan error, 1)
	serverStop := make(chan struct{})
	defer close(serverStop)
	go func() {
		serverErrors <- runTestXMPPServer(listener, certificate, serverStop)
	}()

	config := DefaultConfig("alice@example.org/test", "secret")
	config.Address = listener.Addr().String()
	config.SASLMechanisms = []string{"PLAIN"}
	config.ConnectTimeout = 5 * time.Second
	config.IQTimeout = 5 * time.Second
	config.TLSConfig = &tls.Config{ // #nosec G402 -- local self-signed test server only.
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	messages := make(chan Message, 1)
	client.OnMessage = func(message Message) {
		select {
		case messages <- message:
		default:
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if got, want := client.JID().String(), "alice@example.org/test"; got != want {
		t.Fatalf("bound JID = %q, want %q", got, want)
	}

	select {
	case message := <-messages:
		if message.From != "bob@example.org/phone" || message.Body != "hello over xmpp" {
			t.Fatalf("message = %#v", message)
		}
	case err := <-serverErrors:
		if err != nil {
			t.Fatalf("test server: %v", err)
		}
		t.Fatal("test server exited before delivering a message")
	case <-ctx.Done():
		t.Fatal("timed out waiting for message")
	}

	if err := client.Close(); err != nil && !isClosedNetworkError(err) {
		t.Fatalf("Close: %v", err)
	}
}

func runTestXMPPServer(listener net.Listener, certificate tls.Certificate, stop <-chan struct{}) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	decoder := xml.NewDecoder(conn)
	if _, err := readTestStreamStart(decoder); err != nil {
		return fmt.Errorf("initial stream: %w", err)
	}
	if _, err := io.WriteString(conn, "<stream:stream from='example.org' id='pre-tls' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'><stream:features><starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'><required/></starttls></stream:features>"); err != nil {
		return err
	}
	startTLS, err := readTestNode(decoder)
	if err != nil {
		return fmt.Errorf("read starttls: %w", err)
	}
	if startTLS.Name.Space != startTLSNS || startTLS.Name.Local != "starttls" {
		return fmt.Errorf("expected STARTTLS, got %s", startTLS.MustXML())
	}
	if _, err := io.WriteString(conn, "<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>"); err != nil {
		return err
	}

	tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("TLS handshake: %w", err)
	}
	conn = tlsConn
	decoder = xml.NewDecoder(conn)
	if _, err := readTestStreamStart(decoder); err != nil {
		return fmt.Errorf("post-TLS stream: %w", err)
	}
	if _, err := io.WriteString(conn, "<stream:stream from='example.org' id='post-tls' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'><stream:features><mechanisms xmlns='urn:ietf:params:xml:ns:xmpp-sasl'><mechanism>PLAIN</mechanism></mechanisms></stream:features>"); err != nil {
		return err
	}
	auth, err := readTestNode(decoder)
	if err != nil {
		return fmt.Errorf("read auth: %w", err)
	}
	mechanism, _ := auth.AttrValue("mechanism")
	decodedAuth, err := base64.StdEncoding.DecodeString(auth.Text())
	if err != nil {
		return fmt.Errorf("decode auth: %w", err)
	}
	if mechanism != "PLAIN" || string(decodedAuth) != "\x00alice\x00secret" {
		return fmt.Errorf("unexpected authentication payload")
	}
	if _, err := io.WriteString(conn, "<success xmlns='urn:ietf:params:xml:ns:xmpp-sasl'/>"); err != nil {
		return err
	}

	decoder = xml.NewDecoder(conn)
	if _, err := readTestStreamStart(decoder); err != nil {
		return fmt.Errorf("post-auth stream: %w", err)
	}
	if _, err := io.WriteString(conn, "<stream:stream from='example.org' id='post-auth' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'><stream:features><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/></stream:features>"); err != nil {
		return err
	}
	bindIQ, err := readTestNode(decoder)
	if err != nil {
		return fmt.Errorf("read bind IQ: %w", err)
	}
	id, _ := bindIQ.AttrValue("id")
	if id == "" || bindIQ.Name.Local != "iq" || bindIQ.Child(bindNS, "bind") == nil {
		return fmt.Errorf("invalid bind IQ: %s", bindIQ.MustXML())
	}
	response := "<iq type='result' id='" + escapeXMLAttribute(id) + "'><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'><jid>alice@example.org/test</jid></bind></iq>"
	if _, err := io.WriteString(conn, response); err != nil {
		return err
	}

	presence, err := readTestNode(decoder)
	if err != nil {
		return fmt.Errorf("read initial presence: %w", err)
	}
	if presence.Name.Local != "presence" {
		return fmt.Errorf("expected initial presence, got %s", presence.MustXML())
	}
	if _, err := io.WriteString(conn, "<message from='bob@example.org/phone' to='alice@example.org/test' id='m1' type='chat'><body>hello over xmpp</body></message>"); err != nil {
		return err
	}

	<-stop
	return nil
}

func readTestStreamStart(decoder *xml.Decoder) (xml.StartElement, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return xml.StartElement{}, err
		}
		if start, ok := token.(xml.StartElement); ok {
			if start.Name.Space == streamNS && start.Name.Local == "stream" {
				return start, nil
			}
			return xml.StartElement{}, fmt.Errorf("expected stream, got <%s>", start.Name.Local)
		}
	}
}

func readTestNode(decoder *xml.Decoder) (Node, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return Node{}, err
		}
		if start, ok := token.(xml.StartElement); ok {
			var node Node
			if err := decoder.DecodeElement(&node, &start); err != nil {
				return Node{}, err
			}
			return node, nil
		}
	}
}

func testCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "example.org"},
		DNSNames:     []string{"example.org"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}

func isClosedNetworkError(err error) bool {
	return err == nil || err == net.ErrClosed
}

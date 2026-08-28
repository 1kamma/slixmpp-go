package xmpp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- SCRAM-SHA-1 interoperability requires SHA-1.
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"strconv"
	"strings"
)

const saslNS = "urn:ietf:params:xml:ns:xmpp-sasl"

func offeredSASLMechanisms(features Node) []string {
	mechanisms := features.Child(saslNS, "mechanisms")
	if mechanisms == nil {
		return nil
	}
	var result []string
	for _, child := range mechanisms.Children() {
		if child.Name.Local == "mechanism" {
			result = append(result, strings.TrimSpace(child.Text()))
		}
	}
	return result
}

func selectSASLMechanism(preferred, offered []string, tlsActive, hasCertificate bool, password string) (string, error) {
	available := make(map[string]bool, len(offered))
	for _, m := range offered {
		available[strings.ToUpper(strings.TrimSpace(m))] = true
	}
	for _, m := range preferred {
		m = strings.ToUpper(strings.TrimSpace(m))
		if !available[m] {
			continue
		}
		switch m {
		case "PLAIN":
			if password == "" || !tlsActive {
				continue
			}
		case "EXTERNAL":
			if !hasCertificate || !tlsActive {
				continue
			}
		case "SCRAM-SHA-1", "SCRAM-SHA-256":
			if password == "" {
				continue
			}
		case "ANONYMOUS":
		default:
			continue
		}
		return m, nil
	}
	return "", fmt.Errorf("%w: no mutually supported SASL mechanism (offered %v)", ErrAuthentication, offered)
}

func (c *Client) authenticate(features Node) error {
	offered := offeredSASLMechanisms(features)
	if len(offered) == 0 {
		return fmt.Errorf("%w: server advertised no SASL mechanisms", ErrAuthentication)
	}
	tlsActive := c.tlsActive()
	hasCert := len(c.tlsConfig().Certificates) > 0
	if c.cfg.InsecureAllowPlainAuth && !tlsActive {
		tlsActive = true
	}
	mechanism, err := selectSASLMechanism(c.cfg.SASLMechanisms, offered, tlsActive, hasCert, c.cfg.Password)
	if err != nil {
		return err
	}
	c.debugf("authenticating with SASL %s", mechanism)
	switch mechanism {
	case "PLAIN":
		return c.authPlain()
	case "EXTERNAL":
		return c.authExternal()
	case "ANONYMOUS":
		return c.simpleSASL("ANONYMOUS", nil)
	case "SCRAM-SHA-1":
		return c.authSCRAM(sha1.New, mechanism)
	case "SCRAM-SHA-256":
		return c.authSCRAM(sha256.New, mechanism)
	}
	return fmt.Errorf("%w: unsupported mechanism %s", ErrAuthentication, mechanism)
}
func (c *Client) authPlain() error {
	return c.simpleSASL("PLAIN", []byte(c.cfg.AuthorizationID+"\x00"+c.jid.Local+"\x00"+c.cfg.Password))
}
func (c *Client) authExternal() error {
	payload := c.cfg.AuthorizationID
	if payload == "" {
		payload = c.jid.Bare()
	}
	return c.simpleSASL("EXTERNAL", []byte(payload))
}
func (c *Client) simpleSASL(mechanism string, initial []byte) error {
	auth := NewNode(saslNS, "auth")
	auth.SetAttr("mechanism", mechanism)
	if initial != nil {
		auth.AddText(base64.StdEncoding.EncodeToString(initial))
	}
	if err := c.writeNode(auth); err != nil {
		return err
	}
	response, err := c.readTopLevelNode()
	if err != nil {
		return err
	}
	if response.Name.Space == saslNS && response.Name.Local == "success" {
		return nil
	}
	if response.Name.Space == saslNS && response.Name.Local == "failure" {
		return saslFailure(response)
	}
	return fmt.Errorf("%w: unexpected SASL response <%s>", ErrAuthentication, response.Name.Local)
}

func (c *Client) authSCRAM(hashFactory func() hash.Hash, mechanism string) error {
	nonceBytes := make([]byte, 18)
	if _, err := rand.Read(nonceBytes); err != nil {
		return fmt.Errorf("xmpp: generate SCRAM nonce: %w", err)
	}
	clientNonce := base64.RawStdEncoding.EncodeToString(nonceBytes)
	username := strings.NewReplacer("=", "=3D", ",", "=2C").Replace(c.jid.Local)
	clientFirstBare := "n=" + username + ",r=" + clientNonce
	clientFirst := "n,," + clientFirstBare
	auth := NewTextNode(saslNS, "auth", base64.StdEncoding.EncodeToString([]byte(clientFirst)))
	auth.SetAttr("mechanism", mechanism)
	if err := c.writeNode(auth); err != nil {
		return err
	}
	challenge, err := c.readTopLevelNode()
	if err != nil {
		return err
	}
	if challenge.Name.Local == "failure" {
		return saslFailure(challenge)
	}
	if challenge.Name.Space != saslNS || challenge.Name.Local != "challenge" {
		return fmt.Errorf("%w: expected SCRAM challenge, received <%s>", ErrAuthentication, challenge.Name.Local)
	}
	serverFirstBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(challenge.Text()))
	if err != nil {
		return fmt.Errorf("%w: invalid SCRAM challenge encoding", ErrAuthentication)
	}
	serverFirst := string(serverFirstBytes)
	attributes, err := parseSCRAMAttributes(serverFirst)
	if err != nil {
		return err
	}
	serverNonce := attributes["r"]
	if !strings.HasPrefix(serverNonce, clientNonce) || serverNonce == clientNonce {
		return fmt.Errorf("%w: invalid SCRAM server nonce", ErrAuthentication)
	}
	salt, err := base64.StdEncoding.DecodeString(attributes["s"])
	if err != nil {
		return fmt.Errorf("%w: invalid SCRAM salt", ErrAuthentication)
	}
	iterations, err := strconv.Atoi(attributes["i"])
	if err != nil || iterations < 1 || iterations > 10_000_000 {
		return fmt.Errorf("%w: invalid SCRAM iteration count", ErrAuthentication)
	}
	salted := pbkdf2(hashFactory, []byte(c.cfg.Password), salt, iterations, hashFactory().Size())
	clientKey := hmacDigest(hashFactory, salted, []byte("Client Key"))
	storedKey := digest(hashFactory, clientKey)
	clientFinalNoProof := "c=biws,r=" + serverNonce
	authMessage := clientFirstBare + "," + serverFirst + "," + clientFinalNoProof
	clientSignature := hmacDigest(hashFactory, storedKey, []byte(authMessage))
	proof := xorBytes(clientKey, clientSignature)
	clientFinal := clientFinalNoProof + ",p=" + base64.StdEncoding.EncodeToString(proof)
	serverKey := hmacDigest(hashFactory, salted, []byte("Server Key"))
	expectedSignature := hmacDigest(hashFactory, serverKey, []byte(authMessage))
	response := NewTextNode(saslNS, "response", base64.StdEncoding.EncodeToString([]byte(clientFinal)))
	if err := c.writeNode(response); err != nil {
		return err
	}
	final, err := c.readTopLevelNode()
	if err != nil {
		return err
	}
	if final.Name.Local == "failure" {
		return saslFailure(final)
	}
	if final.Name.Space != saslNS || final.Name.Local != "success" {
		return fmt.Errorf("%w: expected SCRAM success, received <%s>", ErrAuthentication, final.Name.Local)
	}
	if strings.TrimSpace(final.Text()) != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(final.Text()))
		if err != nil {
			return fmt.Errorf("%w: invalid SCRAM server-final encoding", ErrAuthentication)
		}
		attrs, err := parseSCRAMAttributes(string(decoded))
		if err != nil {
			return err
		}
		if serverErr := attrs["e"]; serverErr != "" {
			return fmt.Errorf("%w: SCRAM server error %s", ErrAuthentication, serverErr)
		}
		signature, err := base64.StdEncoding.DecodeString(attrs["v"])
		if err != nil || !hmac.Equal(signature, expectedSignature) {
			return fmt.Errorf("%w: SCRAM server signature mismatch", ErrAuthentication)
		}
	}
	return nil
}

func parseSCRAMAttributes(message string) (map[string]string, error) {
	attrs := make(map[string]string)
	for _, part := range strings.Split(message, ",") {
		if len(part) < 3 || part[1] != '=' {
			return nil, fmt.Errorf("%w: malformed SCRAM attribute", ErrAuthentication)
		}
		key := part[:1]
		if _, ok := attrs[key]; ok {
			return nil, fmt.Errorf("%w: duplicate SCRAM attribute %s", ErrAuthentication, key)
		}
		attrs[key] = part[2:]
	}
	return attrs, nil
}
func pbkdf2(hf func() hash.Hash, password, salt []byte, iterations, keyLength int) []byte {
	hashLength := hf().Size()
	blocks := (keyLength + hashLength - 1) / hashLength
	result := make([]byte, 0, blocks*hashLength)
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(hf, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for iteration := 1; iteration < iterations; iteration++ {
			mac = hmac.New(hf, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for i := range t {
				t[i] ^= u[i]
			}
		}
		result = append(result, t...)
	}
	return result[:keyLength]
}
func hmacDigest(hf func() hash.Hash, key, message []byte) []byte {
	mac := hmac.New(hf, key)
	_, _ = mac.Write(message)
	return mac.Sum(nil)
}
func digest(hf func() hash.Hash, value []byte) []byte {
	h := hf()
	_, _ = h.Write(value)
	return h.Sum(nil)
}
func xorBytes(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}
func saslFailure(node Node) error {
	condition := "failure"
	var text string
	for _, child := range node.Children() {
		if child.Name.Local == "text" {
			text = child.Text()
		} else {
			condition = child.Name.Local
		}
	}
	if text != "" {
		return fmt.Errorf("%w: %s: %s", ErrAuthentication, condition, text)
	}
	return fmt.Errorf("%w: %s", ErrAuthentication, condition)
}

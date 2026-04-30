// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // MD5 required by AMT Digest auth protocol
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/cli/pkg/rest/infra"
)

// ─────────────────────────────────────────────────────────────────────────────
// Embedded Angular app
// ─────────────────────────────────────────────────────────────────────────────

//go:embed static
var staticFiles embed.FS

// spaHandler serves the embedded Angular app with SPA fallback to index.html.
// Any path that doesn't match a real file is redirected to index.html so
// Angular's client-side router can handle it.
type spaHandler struct {
	fileServer http.Handler
	efs        fs.FS
}

func newSPAHandler(efs fs.FS) spaHandler {
	return spaHandler{
		fileServer: http.FileServer(http.FS(efs)),
		efs:        efs,
	}
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	if _, err := h.efs.Open(path); err == nil {
		h.fileServer.ServeHTTP(w, r)
		return
	}
	// Unknown path — let Angular router handle it
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = "/"
	h.fileServer.ServeHTTP(w, r2)
}

// ─────────────────────────────────────────────────────────────────────────────
//  Function for AMT protocol encoding
// ─────────────────────────────────────────────────────────────────────────────

func intToLE32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// hexMD5amt computes an MD5 hex digest.
// MD5 is mandated by the AMT Digest Authentication protocol (RFC 2617 §3.2.2).
// It is NOT used for password storage — it is used as a challenge-response hash
// over a TLS-protected connection, matching the fixed algorithm required by AMT firmware.
// Replacing it with a stronger hash would break compatibility with all AMT devices.
func hexMD5amt(s string) string {
	h := md5.Sum([]byte(s)) //nolint:gosec // MD5 mandated by AMT Digest auth RFC 2617; not used for password storage
	return hex.EncodeToString(h[:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// decodeUTF8Binary - MPS sends binary data in WebSocket frames as UTF-8.
// Bytes 0x80-0xFF are encoded as 2-byte UTF-8 sequences; decode back to raw binary.
func decodeUTF8Binary(src []byte) []byte {
	dst := make([]byte, 0, len(src))
	for i := 0; i < len(src); {
		b := src[i]
		if b < 0x80 {
			dst = append(dst, b)
			i++
		} else if b&0xE0 == 0xC0 && i+1 < len(src) && src[i+1]&0xC0 == 0x80 {
			dst = append(dst, (b&0x1F)<<6|(src[i+1]&0x3F))
			i += 2
		} else {
			dst = append(dst, b)
			i++
		}
	}
	return dst
}

// encodeUTF8Binary - reverse of decodeUTF8Binary.
// Each byte >= 0x80 expands to 2 bytes; capacity hint uses src length.
// append grows the slice automatically if needed — no overflow risk.
func encodeUTF8Binary(src []byte) []byte {
	dst := make([]byte, 0, len(src))
	for _, b := range src {
		if b < 0x80 {
			dst = append(dst, b)
		} else {
			dst = append(dst, 0xC0|(b>>6), 0x80|(b&0x3F))
		}
	}
	return dst
}

// ─────────────────────────────────────────────────────────────────────────────
// KVMSession — AMT protocol + RFB relay
// ─────────────────────────────────────────────────────────────────────────────

type kvmSession struct {
	mpsConn *websocket.Conn
	mpsMu   sync.Mutex

	browserConn          *websocket.Conn
	browserMu            sync.RWMutex
	pendingBrowserFrames [][]byte

	// browserReady is closed once the first browser WebSocket connects.
	// ChannelOpen (0x40) is deferred until then so RFB data flows immediately.
	browserReady chan struct{}

	// AMT state
	amtState    string // "start" → "auth" → "channel" → "active"
	deviceGUID  string
	amtPassword string // empty for CCM; set from AMT_PASSWORD env var for ACM

	state   string // "connecting" | "authenticating" | "active" | "error" | "disconnected"
	stateMu sync.RWMutex

	done chan struct{}
	logf func(string, ...interface{})
}

func (s *kvmSession) setState(v string) {
	s.stateMu.Lock()
	s.state = v
	s.stateMu.Unlock()
}

func (s *kvmSession) getState() string {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

func (s *kvmSession) sendToMPS(data []byte) error {
	s.mpsMu.Lock()
	defer s.mpsMu.Unlock()
	return s.mpsConn.WriteMessage(websocket.TextMessage, encodeUTF8Binary(data))
}

func (s *kvmSession) sendToBrowser(data []byte) {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()
	if s.browserConn == nil {
		copied := make([]byte, len(data))
		copy(copied, data)
		s.pendingBrowserFrames = append(s.pendingBrowserFrames, copied)
		return
	}
	_ = s.browserConn.WriteMessage(websocket.BinaryMessage, data)
}

func (s *kvmSession) flushPending() {
	s.browserMu.Lock()
	defer s.browserMu.Unlock()
	if s.browserConn == nil || len(s.pendingBrowserFrames) == 0 {
		return
	}
	s.logf("[KVM] flushing %d queued frames to browser", len(s.pendingBrowserFrames))
	for _, frame := range s.pendingBrowserFrames {
		if err := s.browserConn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			break
		}
	}
	s.pendingBrowserFrames = nil
}

// readFromMPS handles the MPS WebSocket messages and the AMT protocol state machine.
func (s *kvmSession) readFromMPS() {
	defer func() {
		s.browserMu.Lock()
		if s.browserConn != nil {
			// Send a clean close frame (1001 = going away) so Angular's onclose
			// handler treats this as a server-initiated end-of-session and does
			_ = s.browserConn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "MPS session ended"),
				time.Now().Add(time.Second),
			)
			s.browserConn.Close()
			s.browserConn = nil
		}
		s.browserMu.Unlock()
		// Do NOT close s.done here — session lifecycle is managed only by
		// serveDisconnect (browser Disconnect button) or ctx.Done (Ctrl+C).
		// An MPS read error just means AMT closed the channel; the operator
		// must explicitly stop the session.
	}()

	for {
		msgType, msg, err := s.mpsConn.ReadMessage()
		if err != nil {
			s.logf("[KVM] MPS connection closed: %v (amtState=%s)", err, s.amtState)
			s.setState("error")
			return
		}
		if msgType == websocket.TextMessage {
			msg = decodeUTF8Binary(msg)
		}

		// Route by AMT protocol state, not by browser-visible session state.
		// s.state is set to "active" early (after auth) so the browser can connect
		// quickly via /api/status polling — but the 0x41 ChannelOpenConfirmation
		// must still be handled by handleAMTMessage (which sets s.amtState="active").
		// Only after the channel is truly open do we forward raw bytes to the browser.
		if s.amtState == "active" {
			s.sendToBrowser(msg)
		} else {
			s.handleAMTMessage(msg)
		}
	}
}

func (s *kvmSession) handleAMTMessage(msg []byte) {
	if len(msg) == 0 {
		return
	}
	switch msg[0] {
	case 0x11: // StartRedirectionSessionReply
		if len(msg) < 2 || msg[1] != 0 {
			s.logf("[KVM] AMT session start failed status=%d", msg[1])
			s.setState("error")
			return
		}
		s.logf("[KVM] AMT session started — querying auth methods")
		s.amtState = "auth"
		s.sendAuthQuery()

	case 0x14: // AuthenticateSessionReply
		if len(msg) < 9 {
			return
		}
		status := msg[1]
		authType := msg[4]
		dataLen := binary.LittleEndian.Uint32(msg[5:9])

		s.logf("[KVM] AuthReply status=%d type=%d dataLen=%d", status, authType, dataLen)

		if len(msg) < 9+int(dataLen) {
			return
		}
		authData := msg[9 : 9+dataLen]

		switch {
		case authType == 0 && status == 0:
			// Query response — check if digest (4) available, then send initial
			hasDigest := false
			for _, m := range authData {
				if m == 4 {
					hasDigest = true
					break
				}
			}
			s.logf("[KVM] Auth methods available: %v (hasDigest=%v)", authData, hasDigest)
			s.sendDigestAuthInitial()

		case (authType == 3 || authType == 4) && status == 1:
			// Digest challenge
			s.handleDigestChallenge(msg, authType, dataLen)

		case status == 0 && (authType == 3 || authType == 4):
			// Auth success — signal "active" so the browser can open /ws/kvm.
			// ChannelOpen is deferred until the browser WebSocket is open so that
			// RFB responses (protocol version, security choice…) flow immediately
			// after AMT confirms the channel — preventing AMT's RFB handshake
			// timeout from firing before the browser is ready.
			s.logf("[KVM] Authentication SUCCESS — waiting for browser to connect")
			s.amtState = "channel"
			s.setState("active")
			go func() {
				// If the browser never opens (e.g. no port-forward, closed tab),
				// abort after 10 minutes so the AMT KVM module is not held indefinitely.
				select {
				case <-s.browserReady:
					s.logf("[KVM] Browser connected — opening AMT channel")
					s.sendChannelOpen()
				case <-time.After(10 * time.Minute):
					s.logf("[KVM] Timeout: browser did not connect within 10 minutes — aborting")
					s.setState("error")
					s.close()
				case <-s.done:
				}
			}()

		case authType == 0 && status == 1:
			s.logf("[KVM] Auth query failed — trying digest anyway")
			s.sendDigestAuthInitial()

		default:
			s.logf("[KVM] Auth failed status=%d type=%d", status, authType)
			s.setState("error")
		}

	case 0x41: // ChannelOpenConfirmation
		if len(msg) < 8 {
			s.setState("error")
			return
		}
		s.logf("[KVM] Channel open confirmed — KVM ACTIVE")
		s.amtState = "active"
		s.setState("active")
		if len(msg) > 8 {
			s.sendToBrowser(msg[8:]) // trailing bytes are initial RFB data
		}

	default:
		s.logf("[KVM] Unknown AMT msg type 0x%02x (%d bytes)", msg[0], len(msg))
	}
}

func (s *kvmSession) handleDigestChallenge(msg []byte, authType uint8, dataLen uint32) {
	d := msg[9 : 9+dataLen]
	ptr := 0

	readField := func() (string, bool) {
		if ptr >= len(d) {
			return "", false
		}
		n := int(d[ptr])
		ptr++
		if ptr+n > len(d) {
			return "", false
		}
		v := string(d[ptr : ptr+n])
		ptr += n
		return v, true
	}

	realm, ok := readField()
	if !ok {
		s.setState("error")
		return
	}
	nonce, ok := readField()
	if !ok {
		s.setState("error")
		return
	}
	qop := ""
	if authType == 4 {
		qop, ok = readField()
		if !ok {
			s.setState("error")
			return
		}
	}
	s.logf("[KVM] Digest challenge realm=%s qop=%s", realm, qop)
	s.sendDigestAuthResponse(authType, realm, nonce, qop)
}

// ─── AMT protocol send helpers ────────────────────────────────────────────────

func (s *kvmSession) sendRedirectStartKVM() {
	// [0x10][0x01][0x00][0x00]['K']['V']['M']['R']
	_ = s.sendToMPS([]byte{0x10, 0x01, 0x00, 0x00, 'K', 'V', 'M', 'R'})
	s.logf("[KVM] → RedirectStartKVM sent")
}

func (s *kvmSession) sendAuthQuery() {
	// [0x13][0x00][0x00][0x00][authType=0][authDataLen=0 (4 bytes LE)]
	_ = s.sendToMPS([]byte{0x13, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func (s *kvmSession) sendDigestAuthInitial() {
	user := "admin"
	uri := "/RedirectionService"
	dataLen := len(user) + len(uri) + 8

	var buf bytes.Buffer
	buf.Write([]byte{0x13, 0x00, 0x00, 0x00, 0x04}) // AuthenticateSession, type=4
	buf.Write(intToLE32(uint32(dataLen)))
	buf.WriteByte(byte(len(user)))
	buf.WriteString(user)
	buf.Write([]byte{0x00, 0x00})
	buf.WriteByte(byte(len(uri)))
	buf.WriteString(uri)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})
	_ = s.sendToMPS(buf.Bytes())
	s.logf("[KVM] → DigestAuthInitial sent")
}

func (s *kvmSession) sendDigestAuthResponse(authType uint8, realm, nonce, qop string) {
	user := "admin"
	pass := s.amtPassword // empty for CCM; ACM requires the AMT admin password
	uri := s.deviceGUID
	if uri == "" {
		uri = "/RedirectionService"
	}

	cnonce := randomHex(16)
	nc := "00000001"

	ha1 := hexMD5amt(user + ":" + realm + ":" + pass)
	ha2 := hexMD5amt("POST:" + uri)
	extra := ""
	if authType == 4 {
		extra = nc + ":" + cnonce + ":" + qop + ":"
	}
	digest := hexMD5amt(ha1 + ":" + nonce + ":" + extra + ha2)

	totalLen := len(user) + len(realm) + len(nonce) + len(uri) + len(cnonce) + len(nc) + len(digest) + 7
	if authType == 4 {
		totalLen += len(qop) + 1
	}

	var buf bytes.Buffer
	buf.Write([]byte{0x13, 0x00, 0x00, 0x00, authType})
	buf.Write(intToLE32(uint32(totalLen)))
	for _, field := range []string{user, realm, nonce, uri, cnonce, nc, digest} {
		if len(field) > 255 {
			s.logf("[KVM] Warning: digest field exceeds 255 bytes, truncating")
			field = field[:255]
		}
		buf.WriteByte(byte(len(field)))
		buf.WriteString(field)
	}
	if authType == 4 {
		buf.WriteByte(byte(len(qop)))
		buf.WriteString(qop)
	}

	_ = s.sendToMPS(buf.Bytes())
	s.logf("[KVM] → DigestAuthResponse sent (%d bytes)", buf.Len())
}

func (s *kvmSession) sendChannelOpen() {
	_ = s.sendToMPS([]byte{0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	s.logf("[KVM] → ChannelOpen sent")
}

func (s *kvmSession) keepAlive() {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-t.C:
			s.mpsMu.Lock()
			err := s.mpsConn.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			s.mpsMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (s *kvmSession) readFromBrowser() {
	for {
		_, msg, err := s.browserConn.ReadMessage()
		if err != nil {
			s.browserMu.Lock()
			s.browserConn = nil
			s.browserMu.Unlock()
			return
		}
		if err := s.sendToMPS(msg); err != nil {
			return
		}
	}
}

func (s *kvmSession) close() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	if s.mpsConn != nil {
		s.mpsConn.Close()
	}
	s.setState("disconnected")
}

// ─────────────────────────────────────────────────────────────────────────────
// Local HTTP/WebSocket server
// ─────────────────────────────────────────────────────────────────────────────

type kvmServer struct {
	session  *kvmSession
	upgrader websocket.Upgrader
	mu       sync.RWMutex
}

func (srv *kvmServer) serveStatus(w http.ResponseWriter, _ *http.Request) {
	srv.mu.RLock()
	sess := srv.session
	srv.mu.RUnlock()
	state := "disconnected"
	if sess != nil {
		state = sess.getState()
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"state":%q}`, state)
}

// serveConnect handles POST /api/connect from the Angular app.
// In orch-cli mode the MPS relay and AMT handshake are initiated before the
// browser opens, so the credentials in the request body are not needed — we
// simply acknowledge and let the Angular app proceed to poll /api/status.
func (srv *kvmServer) serveConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	srv.mu.RLock()
	sess := srv.session
	srv.mu.RUnlock()
	state := "connecting"
	if sess != nil {
		state = sess.getState()
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"state":%q}`, state)
}

func (srv *kvmServer) serveDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	srv.mu.RLock()
	sess := srv.session
	srv.mu.RUnlock()
	if sess != nil {
		sess.close()
	}
	w.WriteHeader(http.StatusOK)
}

func (srv *kvmServer) serveKVMWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := srv.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	srv.mu.RLock()
	sess := srv.session
	srv.mu.RUnlock()

	if sess == nil || (sess.getState() == "disconnected" || sess.getState() == "error") {
		// Send a clean close frame so the browser's isCleanClose check fires
		// and it does NOT enter the auto-reconnect loop.
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "session ended"),
			time.Now().Add(time.Second),
		)
		conn.Close()
		return
	}

	sess.browserMu.Lock()
	old := sess.browserConn
	sess.browserConn = conn
	sess.browserMu.Unlock()
	if old != nil {
		old.Close()
	}

	sess.logf("[KVM] Browser WebSocket connected")
	sess.flushPending()

	// Signal the AMT handshake goroutine that the browser is ready.
	// Uses sync.Once semantics via a channel close — safe to call multiple times.
	sess.browserMu.Lock()
	select {
	case <-sess.browserReady:
		// already signalled
	default:
		close(sess.browserReady)
	}
	sess.browserMu.Unlock()

	// No read deadline on the browser WebSocket — browser may be idle for long periods.
	conn.SetPongHandler(func(string) error { return nil })

	// Browser keepalive pinger
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-sess.done:
				return
			case <-t.C:
				sess.browserMu.Lock()
				cur := sess.browserConn
				sess.browserMu.Unlock()
				if cur != conn {
					return
				}
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
					return
				}
			}
		}
	}()

	sess.readFromBrowser()
	sess.logf("[KVM] Browser WebSocket disconnected")
}

// ─────────────────────────────────────────────────────────────────────────────
// mpsRelayTLSConfig builds a tls.Config for the MPS WebSocket dial.
// ─────────────────────────────────────────────────────────────────────────────

func mpsRelayTLSConfig(caPath string, logf func(string, ...interface{})) (*tls.Config, error) {
	if caPath == "" {
		return nil, fmt.Errorf("--orch-ca is required: provide the path to the cluster CA certificate (e.g. orch-ca.crt)")
	}
	caPEM, err := os.ReadFile(caPath) //nolint:gosec // path supplied by operator
	if err != nil {
		return nil, fmt.Errorf("cannot read CA certificate %q: %w", caPath, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("no valid certificates found in %q — ensure it is a valid .crt file", caPath)
	}
	logf("[KVM] TLS: using CA certificate from %q", caPath)
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// connectToMPSRelay — dials the MPS WSS relay and starts the AMT handshake.
// ─────────────────────────────────────────────────────────────────────────────

func connectToMPSRelay(token, mpsDomain, deviceGUID, jwtToken, orchCA string, logf func(string, ...interface{})) (*kvmSession, error) {
	dialURL := fmt.Sprintf("wss://%s/relay/webrelay.ashx?p=2&host=%s&port=16994&tls=0&tls1only=0&mode=kvm",
		mpsDomain, deviceGUID)

	logf("[KVM] Connecting to MPS relay: %s", dialURL)

	tlsConfig, err := mpsRelayTLSConfig(orchCA, logf)
	if err != nil {
		return nil, err
	}
	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
	}
	headers := http.Header{}
	headers.Set("Sec-WebSocket-Protocol", token)
	// Traefik validate-jwt middleware requires the Keycloak JWT in the `jwt` cookie.
	if jwtToken != "" {
		headers.Set("Cookie", "jwt="+jwtToken)
	}

	conn, resp, err := dialer.Dial(dialURL, headers)
	if err != nil {
		extra := ""
		if resp != nil {
			extra = fmt.Sprintf(" (HTTP %d)", resp.StatusCode)
			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				return nil, fmt.Errorf("MPS relay rejected the connection (HTTP %d): a KVM session may already be active on this device.\nStop the existing session first: orch-cli set host <id> --session-state stop", resp.StatusCode)
			}
		}
		return nil, fmt.Errorf("MPS relay dial failed%s: %w", extra, err)
	}
	logf("[KVM] Connected to MPS relay")

	sess := &kvmSession{
		mpsConn:      conn,
		state:        "connecting",
		amtState:     "start",
		deviceGUID:   deviceGUID,
		done:         make(chan struct{}),
		browserReady: make(chan struct{}),
		logf:         logf,
	}

	// No read deadline on the MPS connection — keepAlive pings keep it alive;
	// the session only closes on explicit disconnect or a real network error.
	conn.SetPongHandler(func(string) error { return nil })

	go sess.readFromMPS()
	go sess.keepAlive()

	logf("[KVM] Starting AMT RedirectStart handshake")
	sess.sendRedirectStartKVM()

	return sess, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// startKVMViewer — called from host.go after the relay URL is received.
//
// Flow:
//  1. Connect to MPS relay (AMT handshake starts immediately, frames buffered)
//  2. Start local HTTP server on a random OS-assigned port
//  3. Open browser at http://localhost:{port}
//  4. Block until the session ends (browser disconnect, context cancel, or error)
//  5. Close MPS relay
//  6. PATCH desiredKvmState=KVM_STATE_STOP via inventory API
//  7. Poll until currentKvmState=KVM_STATE_STOP (or timeout)
// ─────────────────────────────────────────────────────────────────────────────

func startKVMViewer(ctx context.Context, token, mpsDomain, deviceGUID, orchCA string, hostClient infra.ClientWithResponsesInterface, projectName, hostID string) error {
	logf := func(format string, args ...interface{}) {
		fmt.Printf(format+"\n", args...)
	}

	// Step 1 — connect to MPS relay
	// Traefik's validate-jwt middleware sits in front of mps-wss; pass the
	// Keycloak access token so the WebSocket upgrade is not rejected with 403.
	jwtToken, err := auth.GetAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token for MPS relay: %w", err)
	}
	// AMT_PASSWORD env var — required for ACM (admin) mode digest auth.
	// Leave unset or empty for CCM mode (no password needed).
	amtPassword := os.Getenv("AMT_PASSWORD")
	if amtPassword != "" {
		logf("[KVM] AMT_PASSWORD set — using for digest auth (ACM mode)")
	} else {
		logf("[KVM] AMT_PASSWORD not set — using empty password (CCM mode)")
	}

	// relayURL carries token+host so connectToMPSRelay can extract them
	sess, err := connectToMPSRelay(token, mpsDomain, deviceGUID, jwtToken, orchCA, logf)
	if err != nil {
		return fmt.Errorf("failed to connect to MPS relay: %w", err)
	}
	sess.amtPassword = amtPassword

	// Step 2 — start local HTTP server on a random OS-assigned port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		sess.close()
		return fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	viewerURL := fmt.Sprintf("http://localhost:%d/?autoconnect=1", port)

	srv := &kvmServer{
		session: sess,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}

	// Build the embedded Angular file system (strip the leading "static" prefix)
	embeddedFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		sess.close()
		return fmt.Errorf("embedded static FS error: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/kvm", srv.serveKVMWebSocket)
	mux.HandleFunc("/api/connect", srv.serveConnect)
	mux.HandleFunc("/api/status", srv.serveStatus)
	mux.HandleFunc("/api/disconnect", srv.serveDisconnect)
	mux.Handle("/", newSPAHandler(embeddedFS))

	httpSrv := &http.Server{
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 0, // streaming — no write timeout
	}

	go func() {
		if err := httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			logf("[KVM] HTTP server error: %v", err)
		}
	}()

	// Step 3 — print URL and open browser.
	fmt.Printf("\n╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  KVM session is LIVE — open this URL in your browser ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n  %s\n\n", viewerURL)
	fmt.Println("Press Ctrl+C to stop the KVM session.")
	go openBrowser(viewerURL)

	// Step 4 — wait for session to end (MPS disconnect, ctx cancel, or browser disconnect)
	select {
	case <-sess.done:
		logf("[KVM] Session ended")
	case <-ctx.Done():
		logf("[KVM] Context cancelled — closing session")
		sess.close()
	}

	// Shutdown local HTTP server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)

	// send KVM_STATE_STOP to inventory
	fmt.Println("Sending KVM_STATE_STOP to inventory...")
	stopState := infra.KVMSTATESTOP
	stopKvmState := stopState

	resp, err := hostClient.HostServiceGetHostWithResponse(ctx, projectName, hostID, auth.AddAuthHeader)
	hostName := ""
	if err == nil && resp.JSON200 != nil {
		hostName = resp.JSON200.Name
	}

	patchBody := infra.HostServicePatchHostJSONRequestBody{
		DesiredKvmState: &stopKvmState,
		Name:            hostName,
	}

	patchResp, err := hostClient.HostServicePatchHostWithResponse(
		ctx, projectName, hostID,
		&infra.HostServicePatchHostParams{},
		patchBody,
		auth.AddAuthHeader,
	)
	if err != nil {
		logf("[KVM] Warning: failed to send KVM_STATE_STOP: %v", err)
	} else if err := checkResponse(patchResp.HTTPResponse, patchResp.Body, "KVM stop"); err != nil {
		logf("[KVM] Warning: KVM stop patch failed: %v", err)
	} else {
		fmt.Println("KVM session stopped.")
	}

	// Step 7 — verify KVM_STATE_STOP
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer verifyCancel()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-verifyCtx.Done():
			logf("[KVM] Timeout waiting for KVM_STATE_STOP confirmation")
			return nil
		case <-ticker.C:
			getResp, err := hostClient.HostServiceGetHostWithResponse(verifyCtx, projectName, hostID, auth.AddAuthHeader)
			if err != nil {
				continue
			}
			if getResp.JSON200 != nil && getResp.JSON200.CurrentKvmState != nil {
				if *getResp.JSON200.CurrentKvmState == infra.KVMSTATESTOP {
					fmt.Println("KVM session closed confirmed.")
					return nil
				}
			} else if getResp.JSON200 != nil && getResp.JSON200.CurrentKvmState == nil {
				// State cleared — treat as stopped
				fmt.Println("KVM session closed confirmed.")
				return nil
			}
		}
	}
}

// openBrowser opens the given URL in the default system browser.
// Supports Linux (xdg-open), macOS (open), and Windows (cmd /c start).
// Errors are silently ignored — the URL is always printed to stdout as fallback.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	_ = cmd.Start()
}

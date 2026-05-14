// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"crypto/md5" //nolint:gosec // MD5 is mandated by AMT HTTP Digest Auth (RFC 2617)
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

// SOLSession manages the AMT SOL protocol state machine over WebSocket
type SOLSession struct {
	// MPS/AMT side
	conn        *websocket.Conn
	connMu      sync.Mutex
	amtSequence uint32
	sequenceMu  sync.Mutex
	solReady    chan struct{}
	amtUser     string
	amtPass     string

	// Flow control: track the sequence number from the last received
	// 0x2A (terminal data) frame from AMT.  This value is sent back
	// in 0x2B heartbeat messages so AMT knows which frames we've
	// consumed and can free its TX buffer to continue sending.
	amtLastRecvSeq uint32
	amtAccMu       sync.Mutex

	// Receive buffer: AMT frames can span multiple WebSocket messages
	// (the MPS relay may split frames at arbitrary boundaries).
	// We accumulate data here and only process complete frames.
	recvBuf []byte

	// Shutdown coordination
	done     chan struct{}
	doneOnce sync.Once

	// Error channel
	errChan chan error
}

// sendACK sends a 0x2B heartbeat carrying the last received 0x2A
// frame sequence number so AMT can free TX buffer and keep sending.
func (s *SOLSession) sendACK() {
	s.amtAccMu.Lock()
	ack := s.amtLastRecvSeq
	s.amtAccMu.Unlock()
	msg := []byte{0x2B, 0x00, 0x00, 0x00}
	msg = append(msg, intToLE(ack)...)
	_ = s.sendBinary(msg)
}

// intToLE writes a uint32 as 4 little-endian bytes.
func intToLE(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// shortToLE writes a uint16 as 2 little-endian bytes.
func shortToLE(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

func (s *SOLSession) nextSequence() uint32 {
	s.sequenceMu.Lock()
	defer s.sequenceMu.Unlock()
	seq := s.amtSequence
	s.amtSequence++
	return seq
}

func (s *SOLSession) sendBinary(data []byte) error {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)) //nolint:errcheck
	return s.conn.WriteMessage(websocket.BinaryMessage, data)
}

// Close closes the SOL session (safe to call multiple times)
func (s *SOLSession) Close() {
	s.doneOnce.Do(func() {
		close(s.done)
	})
}

// sendSOLData wraps terminal input in an AMT SOL data frame (0x28).
func (s *SOLSession) sendSOLData(data string) error {
	seq := s.nextSequence()
	frame := []byte{0x28, 0x00, 0x00, 0x00}
	frame = append(frame, intToLE(seq)...)
	frame = append(frame, shortToLE(uint16(len(data)))...)
	frame = append(frame, []byte(data)...)
	return s.sendBinary(frame)
}

// digestHash computes the MD5 hex-digest required by AMT HTTP Digest
// Authentication (RFC 2617).  MD5 is mandated by the AMT protocol — this is
// a challenge-response mechanism, NOT password storage.  The password is never
// persisted; it is combined with a server-supplied nonce to produce a
// one-time response that proves knowledge of the credential.
func digestHash(str string) string {
	h := md5.Sum([]byte(str)) //nolint:gosec // required by AMT digest auth protocol (RFC 2617)
	return hex.EncodeToString(h[:])
}

func generateRandomNonce(byteLen int) string {
	b := make([]byte, byteLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// createMPSRelayTLSConfig builds a tls.Config for the MPS WebSocket dial.
// It requires the path to the cluster CA certificate (--orch-ca flag).
func createMPSRelayTLSConfig(orchCA string) (*tls.Config, error) {
	if orchCA == "" {
		return nil, fmt.Errorf("--orch-ca is required: provide the path to the cluster CA certificate (e.g. orch-ca.crt)")
	}
	caPEM, err := os.ReadFile(orchCA) //nolint:gosec // path supplied by operator
	if err != nil {
		return nil, fmt.Errorf("cannot read CA certificate %q: %w", orchCA, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("no valid certificates found in %q — ensure it is a valid .crt file", orchCA)
	}
	fmt.Fprintf(os.Stderr, "[SOL] TLS: using CA certificate from %q\n", orchCA)
	return &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool}, nil
}

// sendDigestAuthInitial sends the initial digest auth request (method 4).
func (s *SOLSession) sendDigestAuthInitial() error {
	user := s.amtUser
	uri := ""
	dataLen := uint32(len(user) + len(uri) + 8)
	msg := []byte{0x13, 0x00, 0x00, 0x00, 0x04}
	msg = append(msg, intToLE(dataLen)...)
	msg = append(msg, byte(len(user)))
	msg = append(msg, []byte(user)...)
	msg = append(msg, 0x00, 0x00)
	msg = append(msg, byte(len(uri)))
	msg = append(msg, []byte(uri)...)
	msg = append(msg, 0x00, 0x00, 0x00, 0x00)
	return s.sendBinary(msg)
}

// sendDigestAuthResponse computes and sends the RFC 2617 digest auth response.
func (s *SOLSession) sendDigestAuthResponse(realm, nonce, qop string) error {
	user := s.amtUser
	pass := s.amtPass
	uri := ""
	cnonce := generateRandomNonce(16)
	snc := "00000002"
	ha1 := digestHash(user + ":" + realm + ":" + pass)
	ha2 := digestHash("POST:" + uri)
	responseStr := ha1 + ":" + nonce + ":" + snc + ":" + cnonce + ":" + qop + " :" + ha2
	digest := digestHash(responseStr)

	totalLen := len(user) + len(realm) + len(nonce) + len(uri) +
		len(cnonce) + len(snc) + len(digest) + len(qop) + 8
	msg := []byte{0x13, 0x00, 0x00, 0x00, 0x04}
	msg = append(msg, intToLE(uint32(totalLen))...)
	msg = append(msg, byte(len(user)))
	msg = append(msg, []byte(user)...)
	msg = append(msg, byte(len(realm)))
	msg = append(msg, []byte(realm)...)
	msg = append(msg, byte(len(nonce)))
	msg = append(msg, []byte(nonce)...)
	msg = append(msg, byte(len(uri)))
	msg = append(msg, []byte(uri)...)
	msg = append(msg, byte(len(cnonce)))
	msg = append(msg, []byte(cnonce)...)
	msg = append(msg, byte(len(snc)))
	msg = append(msg, []byte(snc)...)
	msg = append(msg, byte(len(digest)))
	msg = append(msg, []byte(digest)...)
	msg = append(msg, byte(len(qop)))
	msg = append(msg, []byte(qop)...)
	return s.sendBinary(msg)
}

// sendSOLSettings sends SOL configuration (0x20) to the AMT device.
func (s *SOLSession) sendSOLSettings() error {
	seq := s.nextSequence()
	msg := []byte{0x20, 0x00, 0x00, 0x00}
	msg = append(msg, intToLE(seq)...)
	msg = append(msg, shortToLE(10000)...) // MaxTxBuffer
	msg = append(msg, shortToLE(100)...)   // TxTimeout
	msg = append(msg, shortToLE(0)...)     // TxOverflowTimeout
	msg = append(msg, shortToLE(10000)...) // RxTimeout
	msg = append(msg, shortToLE(100)...)   // RxFlushTimeout
	msg = append(msg, shortToLE(0)...)     // Heartbeat
	msg = append(msg, 0x00, 0x00, 0x00, 0x00)
	return s.sendBinary(msg)
}

// handleMPSFrame processes a single AMT frame from the given data slice
// and returns the number of bytes consumed.  Returns 0 when the buffer
// does not yet contain a complete frame (caller must wait for more data).
func (s *SOLSession) handleMPSFrame(data []byte, debug bool) int {
	if len(data) == 0 {
		return 0
	}
	cmd := data[0]

	switch cmd {
	case 0x11: // StartRedirectionSessionReply
		if len(data) < 13 {
			return 0 // incomplete — wait for more data
		}
		status := data[1]
		if status != 0 {
			fmt.Fprintf(os.Stderr, "\nSOL session start failed (status=%d)\n", status)
			return 13 // consume the fixed header even on error
		}
		oemLen := int(data[12])
		frameSize := 13 + oemLen
		if frameSize > len(data) {
			return 0 // incomplete
		}
		if debug {
			fmt.Fprintf(os.Stderr, "[SOL] StartRedirectionSessionReply OK (frame=%d)\n", frameSize)
		}
		authQuery := []byte{0x13, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		_ = s.sendBinary(authQuery)
		return frameSize

	case 0x14: // AuthenticateSessionReply
		if len(data) < 9 {
			return 0 // incomplete
		}
		status := data[1]
		authType := data[4]
		authDataLen := int(binary.LittleEndian.Uint32(data[5:9]))
		frameSize := 9 + authDataLen
		if frameSize > len(data) {
			return 0 // incomplete
		}
		if debug {
			fmt.Fprintf(os.Stderr, "[SOL] AuthReply: status=%d authType=%d dataLen=%d frame=%d\n", status, authType, authDataLen, frameSize)
		}

		if status == 0 && authType == 0 {
			authMethods := data[9 : 9+authDataLen]
			hasDigest := false
			for _, m := range authMethods {
				if m == 4 {
					hasDigest = true
					break
				}
			}
			if hasDigest {
				_ = s.sendDigestAuthInitial()
			} else {
				_ = s.sendSOLSettings()
			}
		} else if status == 0 {
			if debug {
				fmt.Fprintf(os.Stderr, "[SOL] Authentication successful!\n")
			}
			_ = s.sendSOLSettings()
		} else if status == 1 && (authType == 3 || authType == 4) {
			authData := data[9 : 9+authDataLen]
			ptr := 0
			realmLen := int(authData[ptr])
			ptr++
			realm := string(authData[ptr : ptr+realmLen])
			ptr += realmLen
			nonceLen := int(authData[ptr])
			ptr++
			nonce := string(authData[ptr : ptr+nonceLen])
			ptr += nonceLen
			qop := ""
			if authType == 4 && ptr < len(authData) {
				qopLen := int(authData[ptr])
				ptr++
				if ptr+qopLen <= len(authData) {
					qop = string(authData[ptr : ptr+qopLen])
				}
			}
			if debug {
				fmt.Fprintf(os.Stderr, "[SOL] Digest challenge: realm=%q nonce=%q qop=%q\n", realm, nonce, qop)
			}
			_ = s.sendDigestAuthResponse(realm, nonce, qop)
		} else {
			fmt.Fprintf(os.Stderr, "\nSOL authentication failed (status=%d)\n", status)
		}
		return frameSize

	case 0x21: // SOL settings response
		// AMT Start SOL Session Reply layout (23 bytes):
		//   [0]     command (0x21)
		//   [1]     status
		//   [2-3]   OEM error
		//   [4-7]   sequence number
		//   [8-9]   max TX buffer size
		//   [10-11] TX timeout
		//   [12-13] TX overflow timeout
		//   [14-15] host session RX timeout
		//   [16-17] host session flush timeout
		//   [18-19] heartbeat interval
		//   [20-22] reserved (3 bytes)
		const solSettingsReplySize = 23
		if len(data) < solSettingsReplySize {
			return 0 // incomplete
		}
		if debug {
			fmt.Fprintf(os.Stderr, "[SOL] SOL Settings Response (status=%d), sending finalize\n", data[1])
		}
		seq := s.nextSequence()
		finalizeMsg := []byte{0x27, 0x00, 0x00, 0x00}
		finalizeMsg = append(finalizeMsg, intToLE(seq)...)
		finalizeMsg = append(finalizeMsg, 0x00, 0x00, 0x1B, 0x00, 0x00, 0x00)
		_ = s.sendBinary(finalizeMsg)

		// Signal SOL is ready
		select {
		case <-s.solReady:
		default:
			close(s.solReady)
		}
		return solSettingsReplySize

	case 0x24: // EndSOLSession (8 bytes)
		if len(data) < 8 {
			return 0 // incomplete
		}
		if debug {
			fmt.Fprintf(os.Stderr, "[SOL] EndSOLSession received\n")
		}
		return 8

	case 0x29: // Serial settings (10 bytes)
		if len(data) < 10 {
			return 0 // incomplete
		}
		return 10

	case 0x2A: // Incoming terminal data
		if len(data) < 10 {
			return 0 // incomplete — need at least the header
		}
		dataLen := int(binary.LittleEndian.Uint16(data[8:10]))
		frameSize := 10 + dataLen
		if frameSize > len(data) {
			return 0 // incomplete — payload not fully received yet
		}
		if dataLen > 0 {
			// Write terminal data directly to stdout (bypasses fmt buffering)
			os.Stdout.Write(data[10 : 10+dataLen]) //nolint:errcheck
		}

		// Record this frame's sequence number for flow-control ACK.
		// AMT uses the value we send back in 0x2B to determine how much
		// TX buffer it can free. Without this, AMT's buffer fills and
		// it stops sending — causing truncated output.
		seqNum := binary.LittleEndian.Uint32(data[4:8])
		s.amtAccMu.Lock()
		s.amtLastRecvSeq = seqNum
		s.amtAccMu.Unlock()

		// Immediately ACK so AMT frees its TX buffer right away.
		s.sendACK()
		return frameSize

	case 0x2B: // Keep alive (8 bytes) — respond with our ACK byte count
		if len(data) < 8 {
			return 0 // incomplete
		}
		s.sendACK()
		return 8

	default:
		if cmd == 0x00 {
			return 1 // skip NUL padding
		}
		if debug {
			fmt.Fprintf(os.Stderr, "[SOL] Unknown cmd 0x%02X (%d bytes remaining)\n", cmd, len(data))
		}
		// Skip only 1 byte so we don't discard valid frames that follow.
		// The old code returned len(data) which threw away everything.
		return 1
	}
}

// connectSOLSession connects to the MPS relay and runs the AMT SOL protocol
// handshake. The function blocks until Ctrl-C or the MPS connection drops.
func connectSOLSession(token, mpsDomain, deviceGUID, jwtToken, amtPass, orchCA string, _ chan<- int) error {
	// Construct carrier URL so parsed.Host, token and GUID are available below
	sessionURL := fmt.Sprintf("wss://%s/relay/webrelay.ashx?token=%s&host=%s", mpsDomain, token, deviceGUID)
	parsed, err := url.Parse(sessionURL)
	if err != nil {
		return fmt.Errorf("invalid session URL: %w", err)
	}

	redirectToken := parsed.Query().Get("token")
	hostGUID := parsed.Query().Get("host")

	if redirectToken == "" || hostGUID == "" {
		return fmt.Errorf("invalid session URL: missing token or host GUID")
	}

	// Use the redirect token from the session URL provided by sol-manager.
	fmt.Printf("\nConnecting to MPS relay...\n")
	fmt.Printf("  Host: %s\n", parsed.Host)
	fmt.Printf("  GUID: %s\n", hostGUID)
	fmt.Printf("  Using redirect token from sol-manager session URL.\n")

	wsURL := fmt.Sprintf("wss://%s/relay/webrelay.ashx?p=2&host=%s&port=16994&tls=0&tls1only=0&mode=sol",
		parsed.Host, hostGUID)

	// Setup WebSocket dialer with timeout
	tlsConfig, err := createMPSRelayTLSConfig(orchCA)
	if err != nil {
		return err
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
		TLSClientConfig:  tlsConfig,
	}

	headers := http.Header{}
	headers.Add("Sec-WebSocket-Protocol", redirectToken)
	headers.Add("Cookie", fmt.Sprintf("jwt=%s", jwtToken))

	conn, resp, err := dialer.Dial(wsURL, headers)
	if err != nil {
		errMsg := fmt.Sprintf("WebSocket connection failed: %v", err)
		if resp != nil {
			errMsg += fmt.Sprintf(" (HTTP %s)", resp.Status)
			if resp.Body != nil {
				body, _ := io.ReadAll(resp.Body)
				if len(body) > 0 {
					errMsg += fmt.Sprintf(": %s", string(body))
				}
			}
		}
		return fmt.Errorf("%s", errMsg)
	}
	defer conn.Close()

	fmt.Printf("  MPS WebSocket connected!\n")

	// Set up ping/pong handling with proper deadline management
	var readDeadlineMu sync.Mutex
	conn.SetPongHandler(func(_ string) error {
		readDeadlineMu.Lock()
		defer readDeadlineMu.Unlock()
		conn.SetReadDeadline(time.Now().Add(300 * time.Second)) //nolint:errcheck
		return nil
	})
	readDeadlineMu.Lock()
	conn.SetReadDeadline(time.Now().Add(300 * time.Second)) //nolint:errcheck
	readDeadlineMu.Unlock()

	// Initialize SOL session
	sol := &SOLSession{
		conn:     conn,
		solReady: make(chan struct{}),
		amtUser:  "admin",
		amtPass:  amtPass,
		done:     make(chan struct{}),
		errChan:  make(chan error, 5),
	}

	// Send StartRedirectionSession for SOL
	solStartCmd := []byte{0x10, 0x00, 0x00, 0x00, 0x53, 0x4F, 0x4C, 0x20}
	if err := sol.sendBinary(solStartCmd); err != nil {
		return fmt.Errorf("failed to send SOL start: %w", err)
	}

	// Ignore SIGINT so Ctrl+C is passed through to the remote host.
	// Use Ctrl+] to exit the SOL session instead.
	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)

	// Graceful shutdown on SIGTERM only
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	done := make(chan struct{})
	debug := os.Getenv("SOL_DEBUG") != ""

	// Reader goroutine: handles AMT SOL protocol and writes terminal data to stdout
	go func() {
		defer close(done)
		if debug {
			fmt.Fprintf(os.Stderr, "[SOL] Waiting for AMT protocol messages...\n")
		}
		for {
			select {
			case <-sol.done:
				return
			default:
			}

			readDeadlineMu.Lock()
			conn.SetReadDeadline(time.Now().Add(300 * time.Second)) //nolint:errcheck
			readDeadlineMu.Unlock()

			msgType, message, readErr := conn.ReadMessage()
			if readErr != nil {
				if !websocket.IsCloseError(readErr,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived) {
					select {
					case sol.errChan <- fmt.Errorf("connection closed: %w", readErr):
					default:
					}
				}
				return
			}
			if len(message) == 0 {
				continue
			}

			// MPS relay sends AMT protocol frames using BOTH text (type 1)
			// and binary (type 2) WebSocket messages — we must process all
			// of them.  Only skip close/ping/pong control frames.
			if debug && msgType == websocket.TextMessage {
				fmt.Fprintf(os.Stderr, "[SOL] Text WS message (len=%d), processing as AMT frame\n", len(message))
			}

			// Append to receive buffer — AMT frames can span multiple
			// WebSocket messages because the MPS relay may split them
			// at arbitrary byte boundaries.
			sol.recvBuf = append(sol.recvBuf, message...)

			// Process as many complete frames as possible.
			for len(sol.recvBuf) > 0 {
				consumed := sol.handleMPSFrame(sol.recvBuf, debug)
				if consumed <= 0 {
					break // incomplete frame — wait for more data
				}
				sol.recvBuf = sol.recvBuf[consumed:]
			}

			// Prevent unbounded buffer growth from unrecognised padding
			if len(sol.recvBuf) > 64*1024 {
				if debug {
					fmt.Fprintf(os.Stderr, "[SOL] Dropping oversized recvBuf (%d bytes)\n", len(sol.recvBuf))
				}
				sol.recvBuf = nil
			}
		}
	}()

	// Wait for SOL handshake to complete before starting local server
	select {
	case <-sol.solReady:
		// SOL protocol handshake done - send CR to wake terminal
		if err := sol.sendSOLData("\r"); err != nil {
			if debug {
				fmt.Fprintf(os.Stderr, "[SOL] Warning: failed to send wake CR: %v\n", err)
			}
		}
	case <-done:
		// Drain error channel to include the actual failure reason.
		select {
		case connErr := <-sol.errChan:
			return fmt.Errorf("SOL session closed before becoming active: %w", connErr)
		default:
			return fmt.Errorf("SOL session closed before becoming active")
		}
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for SOL session to become active")
	}

	// Start MPS keep-alive pinger (WebSocket ping + SOL empty frame every 10s)
	// Without this, the MPS relay times out and closes the connection.
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-sol.done:
				return
			case <-done:
				return
			case <-ticker.C:
				// WebSocket ping
				sol.connMu.Lock()
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second)) //nolint:errcheck
				pingErr := conn.WriteMessage(websocket.PingMessage, []byte("keepalive"))
				sol.connMu.Unlock()
				if pingErr != nil {
					if debug {
						fmt.Fprintf(os.Stderr, "[SOL] Keep-alive ping failed: %v\n", pingErr)
					}
					return
				}
				// SOL heartbeat (0x2B) carrying the last received frame
				// sequence number.  This is critical for flow control:
				// AMT will stop transmitting if it never sees ACKs.
				sol.amtAccMu.Lock()
				ack := sol.amtLastRecvSeq
				sol.amtAccMu.Unlock()
				frame := []byte{0x2B, 0x00, 0x00, 0x00}
				frame = append(frame, intToLE(ack)...)
				if err := sol.sendBinary(frame); err != nil {
					if debug {
						fmt.Fprintf(os.Stderr, "[SOL] Keep-alive frame failed: %v\n", err)
					}
					return
				}
			}
		}
	}()

	// Print connection info BEFORE setting terminal to raw mode
	// Use \r\n to ensure proper line breaks even if cursor is mid-line from SOL output
	fmt.Printf("\r\n")
	fmt.Printf("========================================\r\n")
	fmt.Printf("  SOL SESSION ACTIVE\r\n")
	fmt.Printf("========================================\r\n")
	fmt.Printf("Press Ctrl+] to disconnect.\r\n")
	fmt.Printf("\r\n")

	// Set terminal to raw mode for direct input
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}

	// Ensure terminal is always restored, even on panic
	defer func() {
		if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: failed to restore terminal: %v\n", err)
		}
	}()

	// Goroutine to read from stdin and send to SOL
	go func() {
		buffer := make([]byte, 1024)
		for {
			select {
			case <-sol.done:
				return
			case <-done:
				return
			default:
			}

			n, readErr := os.Stdin.Read(buffer)
			if readErr != nil {
				if readErr != io.EOF {
					select {
					case sol.errChan <- fmt.Errorf("stdin read error: %w", readErr):
					default:
					}
				}
				return
			}

			if n > 0 {
				if debug {
					fmt.Fprintf(os.Stderr, "[SOL] stdin: %d bytes: %q (hex: %x)\n", n, string(buffer[:n]), buffer[:n])
				}
				// Check for Ctrl+] (0x1D) to exit SOL session.
				// Ctrl+C (0x03) is passed through to the remote host.
				hasExit := false
				for i := 0; i < n; i++ {
					if buffer[i] == 0x1D {
						hasExit = true
						// Send any data before the exit character
						if i > 0 {
							_ = sol.sendSOLData(string(buffer[:i]))
						}
						sol.Close()
						return
					}
				}
				if !hasExit {
					// Send input to SOL
					if err := sol.sendSOLData(string(buffer[:n])); err != nil {
						if debug {
							fmt.Fprintf(os.Stderr, "[SOL] Failed to send data: %v\n", err)
						}
						return
					}
				}
			}
		}
	}()

	// Wait for interrupt, MPS connection close, or Ctrl+C in terminal
	var sessionErr error
	select {
	case <-interrupt:
		if debug {
			fmt.Fprintf(os.Stderr, "\n[SOL] Interrupt signal received\n")
		}
		sol.Close()
	case <-sol.done:
		// Channel already closed by Ctrl+] in terminal
		if debug {
			fmt.Fprintf(os.Stderr, "\n[SOL] Ctrl+] detected in terminal\n")
		}
	case <-done:
		// MPS connection closed
		if debug {
			fmt.Fprintf(os.Stderr, "\n[SOL] MPS connection closed\n")
		}
		sol.Close()
	case sessionErr = <-sol.errChan:
		// Error occurred
		sol.Close()
	}

	// Send WebSocket close message
	sol.connMu.Lock()
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second)) //nolint:errcheck
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	_ = conn.WriteMessage(websocket.CloseMessage, closeMsg)
	sol.connMu.Unlock()

	// Wait a moment for graceful close
	time.Sleep(100 * time.Millisecond)

	conn.Close()

	fmt.Printf("\nSOL session ended.\n")

	return sessionErr
}

// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// SOLSession manages the AMT SOL protocol state machine over WebSocket
// and bridges terminal I/O to a local WebSocket server for wssh3.
type SOLSession struct {
	// MPS/AMT side
	conn        *websocket.Conn
	mu          sync.Mutex
	amtSequence uint32
	solReady    chan struct{}
	amtUser     string
	amtPass     string

	// Browser/wssh3 side
	browserConn *websocket.Conn
	browserMu   sync.Mutex

	// Done channel
	done chan struct{}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	seq := s.amtSequence
	s.amtSequence++
	return seq
}

func (s *SOLSession) sendBinary(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteMessage(websocket.BinaryMessage, data)
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

func hexMD5(str string) string {
	h := md5.Sum([]byte(str))
	return hex.EncodeToString(h[:])
}

func generateRandomNonce(byteLen int) string {
	b := make([]byte, byteLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
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
	ha1 := hexMD5(user + ":" + realm + ":" + pass)
	ha2 := hexMD5("POST:" + uri)
	responseStr := ha1 + ":" + nonce + ":" + snc + ":" + cnonce + ":" + qop + " :" + ha2
	digest := hexMD5(responseStr)

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

// sendToBrowser sends text data to the connected wssh3/browser client.
func (s *SOLSession) sendToBrowser(data string) {
	s.browserMu.Lock()
	bc := s.browserConn
	s.browserMu.Unlock()
	if bc != nil {
		_ = bc.WriteMessage(websocket.TextMessage, []byte(data))
	}
}

// readFromBrowser reads keystrokes from the wssh3/browser WebSocket and
// sends them as AMT SOL data frames to the MPS connection.
func (s *SOLSession) readFromBrowser(conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if len(msg) > 0 {
			_ = s.sendSOLData(string(msg))
		}
	}
}

// connectSOLSession connects to the MPS relay, runs the AMT SOL protocol
// handshake, then starts a local WebSocket proxy server on a random port.
// Users connect via:  wssh3 ws://localhost:<port>/ws/terminal
func connectSOLSession(sessionURL, jwtToken, amtPass string) error {
	// Parse the session URL to extract host, token, guid
	parsed, err := url.Parse(sessionURL)
	if err != nil {
		return fmt.Errorf("invalid session URL: %w", err)
	}

	redirectToken := parsed.Query().Get("token")
	hostGUID := parsed.Query().Get("host")

	// Use the redirect token from the session URL provided by sol-manager.
	fmt.Printf("\nConnecting to MPS relay...\n")
	fmt.Printf("  Host: %s\n", parsed.Host)
	fmt.Printf("  GUID: %s\n", hostGUID)
	fmt.Printf("  Using redirect token from sol-manager session URL.\n")

	wsURL := fmt.Sprintf("wss://%s/relay/webrelay.ashx?p=2&host=%s&port=16994&tls=0&tls1only=0&mode=sol",
		parsed.Host, hostGUID)

	// Setup WebSocket dialer
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		},
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

	// Set up ping/pong handling
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second)) //nolint:errcheck
		return nil
	})
	conn.SetReadDeadline(time.Now().Add(120 * time.Second)) //nolint:errcheck

	// Initialize SOL session
	sol := &SOLSession{
		conn:     conn,
		solReady: make(chan struct{}),
		amtUser:  "admin",
		amtPass:  amtPass,
		done:     make(chan struct{}),
	}

	// Send StartRedirectionSession for SOL
	solStartCmd := []byte{0x10, 0x00, 0x00, 0x00, 0x53, 0x4F, 0x4C, 0x20}
	if err := sol.sendBinary(solStartCmd); err != nil {
		return fmt.Errorf("failed to send SOL start: %w", err)
	}

	// Graceful shutdown on SIGINT
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	done := make(chan struct{})
	debug := os.Getenv("SOL_DEBUG") != ""

	// Reader goroutine: handles AMT SOL protocol + relays terminal data to browser
	go func() {
		defer close(done)
		msgCount := 0
		if debug {
			fmt.Fprintf(os.Stderr, "[SOL] Waiting for AMT protocol messages...\n")
		}
		for {
			conn.SetReadDeadline(time.Now().Add(120 * time.Second)) //nolint:errcheck
			_, message, readErr := conn.ReadMessage()
			if readErr != nil {
				if !websocket.IsCloseError(readErr, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					fmt.Fprintf(os.Stderr, "\nSOL connection closed: %v\n", readErr)
				}
				return
			}
			msgCount++
			if len(message) == 0 {
				continue
			}

			if debug {
				if len(message) <= 64 {
					fmt.Fprintf(os.Stderr, "[SOL] msg#%d cmd=0x%02X len=%d hex=%s\n", msgCount, message[0], len(message), hex.EncodeToString(message))
				} else {
					fmt.Fprintf(os.Stderr, "[SOL] msg#%d cmd=0x%02X len=%d hex=%s...\n", msgCount, message[0], len(message), hex.EncodeToString(message[:64]))
				}
			}

			switch message[0] {
			case 0x11: // StartRedirectionSessionReply
				if len(message) < 4 || message[1] != 0 {
					fmt.Fprintf(os.Stderr, "\nSOL session start failed (status=%d)\n", message[1])
					return
				}
				if debug {
					fmt.Fprintf(os.Stderr, "[SOL] StartRedirectionSessionReply OK\n")
				}
				authQuery := []byte{0x13, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
				_ = sol.sendBinary(authQuery)

			case 0x14: // AuthenticateSessionReply
				if len(message) < 9 {
					continue
				}
				status := message[1]
				authType := message[4]
				authDataLen := int(binary.LittleEndian.Uint32(message[5:9]))
				if debug {
					fmt.Fprintf(os.Stderr, "[SOL] AuthReply: status=%d authType=%d dataLen=%d\n", status, authType, authDataLen)
				}

				if status == 0 && authType == 0 {
					var authMethods []byte
					if len(message) >= 9+authDataLen {
						authMethods = message[9 : 9+authDataLen]
					}
					hasDigest := false
					for _, m := range authMethods {
						if m == 4 {
							hasDigest = true
							break
						}
					}
					if hasDigest {
						_ = sol.sendDigestAuthInitial()
					} else {
						_ = sol.sendSOLSettings()
					}
				} else if status == 0 {
					_ = sol.sendSOLSettings()
				} else if status == 1 && (authType == 3 || authType == 4) {
					if len(message) < 9+authDataLen {
						continue
					}
					authData := message[9 : 9+authDataLen]
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
					_ = sol.sendDigestAuthResponse(realm, nonce, qop)
				} else {
					fmt.Fprintf(os.Stderr, "\nSOL authentication failed (status=%d)\n", status)
					return
				}

			case 0x21: // SOL settings response — finalize session
				if debug {
					fmt.Fprintf(os.Stderr, "[SOL] SOL Settings Response, sending finalize\n")
				}
				seq := sol.nextSequence()
				finalizeMsg := []byte{0x27, 0x00, 0x00, 0x00}
				finalizeMsg = append(finalizeMsg, intToLE(seq)...)
				finalizeMsg = append(finalizeMsg, 0x00, 0x00, 0x1B, 0x00, 0x00, 0x00)
				_ = sol.sendBinary(finalizeMsg)

				// Signal SOL is ready
				select {
				case <-sol.solReady:
				default:
					close(sol.solReady)
				}

			case 0x2A: // Incoming terminal data → relay to browser/wssh3
				if len(message) < 10 {
					continue
				}
				dataLen := int(message[8]) | int(message[9])<<8
				if len(message) < 10+dataLen {
					dataLen = len(message) - 10
				}
				termData := string(message[10 : 10+dataLen])
				sol.sendToBrowser(termData)

			case 0x29: // Serial settings — ignore
			case 0x2B: // Keep alive — ignore
			default:
				if debug {
					fmt.Fprintf(os.Stderr, "[SOL] Unknown cmd 0x%02X (%d bytes)\n", message[0], len(message))
				}
			}
		}
	}()

	// Wait for SOL handshake to complete before starting local server
	select {
	case <-sol.solReady:
		// SOL protocol handshake done
	case <-done:
		return fmt.Errorf("SOL session closed before becoming active")
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for SOL session to become active")
	}

	// Start MPS keep-alive pinger (WebSocket ping + SOL empty frame every 15s)
	// Without this, the MPS relay times out and closes the connection.
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-sol.done:
				return
			case <-done:
				return
			case <-ticker.C:
				// WebSocket ping
				sol.mu.Lock()
				pingErr := conn.WriteMessage(websocket.PingMessage, []byte("keepalive"))
				sol.mu.Unlock()
				if pingErr != nil {
					return
				}
				// SOL keepalive frame (0x28 with 0-length data)
				seq := sol.nextSequence()
				frame := []byte{0x28, 0x00, 0x00, 0x00}
				frame = append(frame, intToLE(seq)...)
				frame = append(frame, shortToLE(0)...)
				_ = sol.sendBinary(frame)
			}
		}
	}()

	// =====================================================================
	// Start local WebSocket proxy server for wssh3
	// =====================================================================
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start local server: %w", err)
	}
	defer listener.Close()
	localPort := listener.Addr().(*net.TCPAddr).Port

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mux := http.NewServeMux()

	// /ws/terminal — WebSocket endpoint for wssh3
	mux.HandleFunc("/ws/terminal", func(w http.ResponseWriter, r *http.Request) {
		wsConn, upgradeErr := upgrader.Upgrade(w, r, nil)
		if upgradeErr != nil {
			fmt.Fprintf(os.Stderr, "[SOL] WebSocket upgrade failed: %v\n", upgradeErr)
			return
		}

		// Attach as browser connection
		sol.browserMu.Lock()
		oldConn := sol.browserConn
		sol.browserConn = wsConn
		sol.browserMu.Unlock()
		if oldConn != nil {
			oldConn.Close()
		}

		// Send CR to wake terminal
		_ = sol.sendSOLData("\r")

		// Browser ping/pong keepalive
		wsConn.SetReadDeadline(time.Now().Add(120 * time.Second)) //nolint:errcheck
		wsConn.SetPongHandler(func(appData string) error {
			wsConn.SetReadDeadline(time.Now().Add(120 * time.Second)) //nolint:errcheck
			return nil
		})

		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-sol.done:
					return
				case <-done:
					return
				case <-ticker.C:
					sol.browserMu.Lock()
					currentConn := sol.browserConn
					sol.browserMu.Unlock()
					if currentConn != wsConn {
						return
					}
					if pingErr := wsConn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); pingErr != nil {
						return
					}
				}
			}
		}()

		// Read keystrokes from wssh3 → send to AMT
		sol.readFromBrowser(wsConn)
	})

	// /api/status — simple status endpoint
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"state":"active","device":"%s","mpsHost":"%s"}`, hostGUID, parsed.Host)
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if srvErr := srv.Serve(listener); srvErr != nil && srvErr != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "[SOL] Local server error: %v\n", srvErr)
		}
	}()

	// Print connection info
	fmt.Printf("\n========================================\n")
	fmt.Printf("  SOL SESSION ACTIVE\n")
	fmt.Printf("========================================\n")
	fmt.Printf("\nConnect with wssh3:\n")
	fmt.Printf("  wssh3 ws://localhost:%d/ws/terminal\n\n", localPort)
	fmt.Printf("Or with websocat:\n")
	fmt.Printf("  websocat ws://localhost:%d/ws/terminal\n\n", localPort)
	fmt.Printf("Press Ctrl+C to disconnect.\n\n")

	// Wait for interrupt or MPS connection close
	select {
	case <-interrupt:
		fmt.Printf("\n\nDisconnecting SOL session...\n")
	case <-done:
	}

	close(sol.done)
	conn.Close()
	srv.Close()
	fmt.Printf("SOL session ended.\n")
	return nil
}

// getAMTPassword retrieves the AMT password. It checks the AMT_PASSWORD
// environment variable first, then uses a hardcoded fallback for development.
func getAMTPassword() string {
	if pass := os.Getenv("AMT_PASSWORD"); pass != "" {
		return pass
	}
	// Try to read from common secret locations
	// In production, this would come from Vault or K8s secret
	return ""
}

// getJWTToken retrieves the current JWT access token from the auth store.
func getJWTTokenFromEnv() string {
	if token := os.Getenv("JWT_TOKEN"); token != "" {
		return token
	}
	return ""
}

// parseSessionURLHost extracts the host portion from a wss:// session URL
func parseSessionURLHost(sessionURL string) string {
	parsed, err := url.Parse(sessionURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

// isSOLSessionURL checks if a string looks like a valid SOL session URL
func isSOLSessionURL(s string) bool {
	return strings.HasPrefix(s, "wss://") && strings.Contains(s, "webrelay.ashx")
}

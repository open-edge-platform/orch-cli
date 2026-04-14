// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package sol implements the AMT SOL (Serial-Over-LAN) WebSocket client
// for orch-cli. It connects directly to the MPS relay, performs the
// AMT redirection handshake (digest auth + SOL settings), and exposes
// an interactive terminal session.
package sol

import (
"crypto/md5"
"crypto/rand"
"crypto/tls"
"encoding/binary"
"encoding/hex"
"fmt"
"io"
"net/http"
"os"
"sync"

"github.com/gorilla/websocket"
"golang.org/x/term"
)

// SOLSessionInfo is the JSON structure received from sol_session_url in inventory.
// sol-manager writes this blob; orch-cli parses and uses it.
type SOLSessionInfo struct {
MPSHost       string `json:"mpsHost"`
DeviceGUID    string `json:"deviceGUID"`
Port          int    `json:"port"`
RedirectToken string `json:"redirectToken"`
KeycloakToken string `json:"keycloakToken"`
AMTUser       string `json:"amtUser"`
AMTPass       string `json:"amtPass"`
}

// SOLSession manages the AMT SOL protocol state machine over a
// WebSocket connection to the MPS relay.
type SOLSession struct {
conn        *websocket.Conn
mu          sync.Mutex
amtSequence uint32
SolReady    chan struct{} // closed when SOL handshake is complete
Done        chan struct{} // closed when session ends
stopOnce    sync.Once
user        string
pass        string
authURI     string
}

// ----- helper functions -----

func intToLE(v uint32) []byte {
b := make([]byte, 4)
binary.LittleEndian.PutUint32(b, v)
return b
}

func shortToLE(v uint16) []byte {
b := make([]byte, 2)
binary.LittleEndian.PutUint16(b, v)
return b
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

// ----- SOLSession methods -----

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

// SendSOLData sends terminal input as an AMT SOL data frame (0x28).
func (s *SOLSession) SendSOLData(data []byte) error {
seq := s.nextSequence()
frame := []byte{0x28, 0x00, 0x00, 0x00}
frame = append(frame, intToLE(seq)...)
frame = append(frame, shortToLE(uint16(len(data)))...)
frame = append(frame, data...)
return s.sendBinary(frame)
}

// Close tears down the WebSocket connection and signals Done.
func (s *SOLSession) Close() {
s.stopOnce.Do(func() {
if s.conn != nil {
_ = s.conn.Close()
}
select {
case <-s.Done:
default:
close(s.Done)
}
})
}

// ----- Auth helpers -----

func (s *SOLSession) sendDigestAuthInitial() error {
user := s.user
uri := s.authURI
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

func (s *SOLSession) sendDigestAuthResponse(realm, nonce, qop string) error {
user := s.user
pass := s.pass
uri := s.authURI
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

// ----- Connect & Handshake -----

// Connect dials the MPS relay WebSocket and performs the AMT SOL
// protocol handshake. It returns once the WebSocket is connected and
// the handshake goroutine is running. Wait on session.SolReady to
// know when the terminal is active, or session.Done to detect disconnect.
func Connect(info SOLSessionInfo, insecure bool) (*SOLSession, error) {
if info.Port == 0 {
info.Port = 16994
}

wsURL := fmt.Sprintf(
"wss://%s/relay/webrelay.ashx?p=2&host=%s&port=%d&tls=0&tls1only=0&mode=sol",
info.MPSHost, info.DeviceGUID, info.Port)

fmt.Fprintf(os.Stderr, "Connecting to MPS relay %s (device=%s)...\n",
info.MPSHost, info.DeviceGUID)

dialer := websocket.Dialer{}
if insecure {
dialer.TLSClientConfig = &tls.Config{
InsecureSkipVerify: true, //nolint:gosec // insecure for development
}
}

headers := http.Header{}
headers.Add("Sec-WebSocket-Protocol", info.RedirectToken)
if info.KeycloakToken != "" {
headers.Add("Cookie", fmt.Sprintf("jwt=%s", info.KeycloakToken))
}

conn, resp, err := dialer.Dial(wsURL, headers)
if err != nil {
errMsg := fmt.Sprintf("WebSocket dial failed: %v", err)
if resp != nil {
body, _ := io.ReadAll(resp.Body)
errMsg += fmt.Sprintf(" (HTTP %s: %s)", resp.Status, string(body))
}
return nil, fmt.Errorf("%s", errMsg)
}

session := &SOLSession{
conn:     conn,
SolReady: make(chan struct{}),
Done:     make(chan struct{}),
user:     info.AMTUser,
pass:     info.AMTPass,
}

// Send StartRedirectionSession for SOL: 0x10 "SOL "
solStartCmd := []byte{0x10, 0x00, 0x00, 0x00, 0x53, 0x4F, 0x4C, 0x20}
if err := session.sendBinary(solStartCmd); err != nil {
conn.Close()
return nil, fmt.Errorf("failed to send SOL start: %w", err)
}

// Start protocol reader goroutine
go session.protocolReader()

return session, nil
}

// protocolReader handles the full AMT SOL protocol state machine.
func (s *SOLSession) protocolReader() {
defer func() {
select {
case <-s.Done:
default:
close(s.Done)
}
}()

for {
_, message, err := s.conn.ReadMessage()
if err != nil {
if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
fmt.Fprintln(os.Stderr, "\nSOL session closed by remote.")
} else {
fmt.Fprintf(os.Stderr, "\nSOL connection error: %v\n", err)
}
return
}

if len(message) == 0 {
continue
}

switch message[0] {
case 0x11: // StartRedirectionSessionReply
if len(message) < 4 || message[1] != 0 {
fmt.Fprintf(os.Stderr, "SOL start failed (status=%d)\n", message[1])
return
}
authQuery := []byte{0x13, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
if err := s.sendBinary(authQuery); err != nil {
fmt.Fprintf(os.Stderr, "Failed to send auth query: %v\n", err)
return
}

case 0x14: // AuthenticateSessionReply
if len(message) < 9 {
continue
}
status := message[1]
authType := message[4]
authDataLen := int(binary.LittleEndian.Uint32(message[5:9]))

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
if err := s.sendDigestAuthInitial(); err != nil {
fmt.Fprintf(os.Stderr, "Digest auth initial failed: %v\n", err)
return
}
} else {
if err := s.sendSOLSettings(); err != nil {
fmt.Fprintf(os.Stderr, "Send SOL settings failed: %v\n", err)
return
}
}
} else if status == 0 {
if err := s.sendSOLSettings(); err != nil {
fmt.Fprintf(os.Stderr, "Send SOL settings failed: %v\n", err)
return
}
} else if status == 1 && (authType == 3 || authType == 4) {
if len(message) < 9+authDataLen {
fmt.Fprintln(os.Stderr, "Auth challenge too short")
return
}
authData := message[9 : 9+authDataLen]
curPtr := 0
realmLen := int(authData[curPtr])
curPtr++
realm := string(authData[curPtr : curPtr+realmLen])
curPtr += realmLen
nonceLen := int(authData[curPtr])
curPtr++
nonce := string(authData[curPtr : curPtr+nonceLen])
curPtr += nonceLen
qop := ""
if authType == 4 && curPtr < len(authData) {
qopLen := int(authData[curPtr])
curPtr++
if curPtr+qopLen <= len(authData) {
qop = string(authData[curPtr : curPtr+qopLen])
}
}
if err := s.sendDigestAuthResponse(realm, nonce, qop); err != nil {
fmt.Fprintf(os.Stderr, "Digest auth response failed: %v\n", err)
return
}
} else {
fmt.Fprintf(os.Stderr, "AMT authentication failed: status=%d authType=%d\n",
status, authType)
return
}

case 0x21: // SOL Settings Response
seq := s.nextSequence()
finalizeMsg := []byte{0x27, 0x00, 0x00, 0x00}
finalizeMsg = append(finalizeMsg, intToLE(seq)...)
finalizeMsg = append(finalizeMsg, 0x00, 0x00, 0x1B, 0x00, 0x00, 0x00)
if err := s.sendBinary(finalizeMsg); err != nil {
fmt.Fprintf(os.Stderr, "Finalize failed: %v\n", err)
return
}
select {
case <-s.SolReady:
default:
close(s.SolReady)
}

case 0x29: // Serial Settings — ignore

case 0x2A: // Incoming display data (terminal output from AMT)
if len(message) < 10 {
continue
}
dataLen := int(message[8]) | int(message[9])<<8
if len(message) < 10+dataLen {
dataLen = len(message) - 10
}
termData := message[10 : 10+dataLen]
_, _ = os.Stdout.Write(termData)

case 0x2B: // Keep alive — ignore

default:
// Unknown command — ignore
}
}
}

// RunInteractive runs an interactive SOL terminal session.
// It puts the terminal in raw mode, forwards keystrokes to the AMT device,
// and prints received output to stdout. Press Ctrl+] to disconnect.
//
// This blocks until the session ends (Ctrl+] or remote disconnect).
func RunInteractive(session *SOLSession) error {
select {
case <-session.SolReady:
// good
case <-session.Done:
return fmt.Errorf("SOL session ended before handshake completed")
}

fmt.Fprintln(os.Stderr, "SOL session active. Press Ctrl+] to disconnect.")

oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
if err != nil {
return fmt.Errorf("failed to set terminal raw mode: %w", err)
}
defer func() {
_ = term.Restore(int(os.Stdin.Fd()), oldState)
fmt.Fprintln(os.Stderr, "\nSOL session ended.")
}()

go func() {
buf := make([]byte, 256)
for {
n, readErr := os.Stdin.Read(buf)
if readErr != nil {
session.Close()
return
}
for i := 0; i < n; i++ {
if buf[i] == 0x1D { // Ctrl+]
fmt.Fprintln(os.Stderr, "\nCtrl+] pressed, disconnecting...")
session.Close()
return
}
}
if sendErr := session.SendSOLData(buf[:n]); sendErr != nil {
fmt.Fprintf(os.Stderr, "\nSend error: %v\n", sendErr)
session.Close()
return
}
}
}()

<-session.Done
return nil
}

// BuildWSURL constructs the MPS relay WebSocket URL from session info.
func BuildWSURL(info SOLSessionInfo) string {
port := info.Port
if port == 0 {
port = 16994
}
return fmt.Sprintf(
"wss://%s/relay/webrelay.ashx?p=2&host=%s&port=%d&tls=0&tls1only=0&mode=sol",
info.MPSHost, info.DeviceGUID, port)
}

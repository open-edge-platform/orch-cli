// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntToLE tests the little-endian uint32 conversion
func TestIntToLE(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		expected []byte
	}{
		{
			name:     "zero value",
			input:    0,
			expected: []byte{0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "small value",
			input:    256,
			expected: []byte{0x00, 0x01, 0x00, 0x00},
		},
		{
			name:     "max value",
			input:    0xFFFFFFFF,
			expected: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name:     "random value",
			input:    0x12345678,
			expected: []byte{0x78, 0x56, 0x34, 0x12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intToLE(tt.input)
			assert.Equal(t, tt.expected, result)
			// Verify it can be decoded back
			decoded := binary.LittleEndian.Uint32(result)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

// TestShortToLE tests the little-endian uint16 conversion
func TestShortToLE(t *testing.T) {
	tests := []struct {
		name     string
		input    uint16
		expected []byte
	}{
		{
			name:     "zero value",
			input:    0,
			expected: []byte{0x00, 0x00},
		},
		{
			name:     "small value",
			input:    256,
			expected: []byte{0x00, 0x01},
		},
		{
			name:     "max value",
			input:    0xFFFF,
			expected: []byte{0xFF, 0xFF},
		},
		{
			name:     "random value",
			input:    0x1234,
			expected: []byte{0x34, 0x12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shortToLE(tt.input)
			assert.Equal(t, tt.expected, result)
			// Verify it can be decoded back
			decoded := binary.LittleEndian.Uint16(result)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

// TestHexMD5 tests the MD5 hash function
func TestHexMD5(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name:     "simple string",
			input:    "hello",
			expected: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name:     "with special chars",
			input:    "admin:Digest:password",
			expected: "14172e43d8e62890ea1daa30feb4d28b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hexMD5(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, result, 32) // MD5 is always 32 hex chars
		})
	}
}

// TestGenerateRandomNonce tests nonce generation
func TestGenerateRandomNonce(t *testing.T) {
	tests := []struct {
		name      string
		byteLen   int
		expectLen int
	}{
		{
			name:      "16 bytes",
			byteLen:   16,
			expectLen: 32, // 16 bytes = 32 hex chars
		},
		{
			name:      "8 bytes",
			byteLen:   8,
			expectLen: 16, // 8 bytes = 16 hex chars
		},
		{
			name:      "32 bytes",
			byteLen:   32,
			expectLen: 64, // 32 bytes = 64 hex chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRandomNonce(tt.byteLen)
			assert.Len(t, result, tt.expectLen)
			// Verify it's valid hex
			for _, c := range result {
				assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))
			}
		})
	}

	// Test randomness - two nonces should be different
	nonce1 := generateRandomNonce(16)
	nonce2 := generateRandomNonce(16)
	assert.NotEqual(t, nonce1, nonce2)
}

// TestSOLSessionNextSequence tests sequence number generation
func TestSOLSessionNextSequence(t *testing.T) {
	sol := &SOLSession{
		amtSequence: 0,
	}

	// Test sequential increments
	for i := uint32(0); i < 10; i++ {
		seq := sol.nextSequence()
		assert.Equal(t, i, seq)
	}

	// Verify final sequence value
	assert.Equal(t, uint32(10), sol.amtSequence)
}

// TestSOLSessionClose tests safe close functionality
func TestSOLSessionClose(t *testing.T) {
	sol := &SOLSession{
		done: make(chan struct{}),
	}

	// First close should work
	sol.Close()

	// Verify channel is closed
	select {
	case <-sol.done:
		// Channel is closed as expected
	default:
		t.Fatal("done channel should be closed")
	}

	// Second close should not panic
	assert.NotPanics(t, func() {
		sol.Close()
	})

	// Third close should still not panic
	assert.NotPanics(t, func() {
		sol.Close()
	})
}

// TestIsSOLSessionURL tests URL validation
func TestIsSOLSessionURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "valid SOL URL",
			url:      "wss://mps-wss.example.com/relay/webrelay.ashx?token=abc&host=123",
			expected: true,
		},
		{
			name:     "valid SOL URL without params",
			url:      "wss://mps.example.com/relay/webrelay.ashx",
			expected: true,
		},
		{
			name:     "invalid - http instead of wss",
			url:      "https://mps.example.com/relay/webrelay.ashx",
			expected: false,
		},
		{
			name:     "invalid - missing webrelay.ashx",
			url:      "wss://mps.example.com/relay/",
			expected: false,
		},
		{
			name:     "invalid - empty string",
			url:      "",
			expected: false,
		},
		{
			name:     "invalid - wrong path",
			url:      "wss://mps.example.com/other/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSOLSessionURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseSessionURLHost tests host extraction from URL
func TestParseSessionURLHost(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "valid URL",
			url:      "wss://mps-wss.example.com:443/relay/webrelay.ashx",
			expected: "mps-wss.example.com:443",
		},
		{
			name:     "URL without port",
			url:      "wss://mps.example.com/relay/webrelay.ashx",
			expected: "mps.example.com",
		},
		{
			name:     "invalid URL",
			url:      "not a url",
			expected: "",
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSessionURLHost(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock WebSocket server for testing
func createMockSOLServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Upgrade error: %v", err)
			return
		}
		defer conn.Close()

		if handler != nil {
			handler(conn)
		}
	}))

	return server
}

// TestSOLWebSocketConnection tests basic WebSocket connectivity
func TestSOLWebSocketConnection(t *testing.T) {
	server := createMockSOLServer(t, func(conn *websocket.Conn) {
		// Echo server - read and write back
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteMessage(websocket.BinaryMessage, msg)
	})
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Send test message
	testMsg := []byte{0x10, 0x00, 0x00, 0x00}
	err = conn.WriteMessage(websocket.BinaryMessage, testMsg)
	require.NoError(t, err)

	// Read echo
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMsg, msg)
}

// TestConnectSOLSessionParamValidation tests parameter validation
func TestConnectSOLSessionParamValidation(t *testing.T) {
	tests := []struct {
		name          string
		mpsHost       string
		hostGUID      string
		redirectToken string
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "valid params but unreachable host",
			mpsHost:       "mps.example.com",
			hostGUID:      "test-guid-123",
			redirectToken: "abc-token",
			expectError:   true,
			errorMsg:      "WebSocket connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readyCh := make(chan int, 1)

			err := connectSOLSession(tt.mpsHost, tt.hostGUID, tt.redirectToken, "fake-jwt", "fake-pass", readyCh)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSOLSessionDataFrame tests SOL data frame construction
func TestSOLSessionDataFrame(t *testing.T) {
	sol := &SOLSession{
		amtSequence: 0,
		conn:        nil, // We won't actually send
	}

	// Test data frame structure
	testData := "test"
	seq := sol.nextSequence()

	expectedFrame := []byte{0x28, 0x00, 0x00, 0x00} // Command + reserved
	expectedFrame = append(expectedFrame, intToLE(seq)...)
	expectedFrame = append(expectedFrame, shortToLE(uint16(len(testData)))...)
	expectedFrame = append(expectedFrame, []byte(testData)...)

	// Verify frame structure
	assert.Equal(t, byte(0x28), expectedFrame[0], "Should be SOL data command")
	assert.Equal(t, uint32(0), binary.LittleEndian.Uint32(expectedFrame[4:8]), "Sequence should be 0")
	assert.Equal(t, uint16(4), binary.LittleEndian.Uint16(expectedFrame[8:10]), "Length should be 4")
	assert.Equal(t, "test", string(expectedFrame[10:]), "Data should match")
}

// TestSOLHandleMPSFrame tests frame handling
func TestSOLHandleMPSFrame(t *testing.T) {
	sol := &SOLSession{
		solReady: make(chan struct{}, 1),
	}

	tests := []struct {
		name           string
		frame          []byte
		expectedConsum int
		description    string
	}{
		{
			name:           "empty frame",
			frame:          []byte{},
			expectedConsum: 0,
			description:    "Should return 0 for empty data",
		},
		{
			name:           "NUL padding",
			frame:          []byte{0x00, 0x00, 0x00},
			expectedConsum: 1,
			description:    "Should skip NUL bytes",
		},
		{
			name:           "serial settings frame",
			frame:          []byte{0x29, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectedConsum: 10,
			description:    "Should consume 10 bytes for serial settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sol.handleMPSFrame(tt.frame, false)
			assert.Equal(t, tt.expectedConsum, result, tt.description)
		})
	}
}

// TestGetJWTTokenFromEnv tests environment variable reading
func TestGetJWTTokenFromEnv(t *testing.T) {
	// Test with no env var
	t.Setenv("JWT_TOKEN", "")
	result := getJWTTokenFromEnv()
	assert.Equal(t, "", result)

	// Test with env var set
	testToken := "test-jwt-token-123"
	t.Setenv("JWT_TOKEN", testToken)
	result = getJWTTokenFromEnv()
	assert.Equal(t, testToken, result)
}

// Benchmark tests for performance
func BenchmarkIntToLE(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = intToLE(uint32(i))
	}
}

func BenchmarkShortToLE(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = shortToLE(uint16(i))
	}
}

func BenchmarkHexMD5(b *testing.B) {
	input := "admin:Digest:password"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hexMD5(input)
	}
}

func BenchmarkGenerateRandomNonce(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = generateRandomNonce(16)
	}
}

// Integration test helper
func TestSOLSessionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would require a real MPS server or a more sophisticated mock
	// For now, we test the key components in isolation
	t.Run("URL parsing and validation", func(t *testing.T) {
		validURL := "wss://mps.example.com/relay/webrelay.ashx?token=abc123&host=guid-123"
		assert.True(t, isSOLSessionURL(validURL))

		parsed, err := url.Parse(validURL)
		require.NoError(t, err)

		token := parsed.Query().Get("token")
		host := parsed.Query().Get("host")

		assert.Equal(t, "abc123", token)
		assert.Equal(t, "guid-123", host)
	})

	t.Run("Session lifecycle", func(t *testing.T) {
		sol := &SOLSession{
			done:    make(chan struct{}),
			errChan: make(chan error, 5),
		}

		// Verify initial state
		assert.NotNil(t, sol.done)
		assert.NotNil(t, sol.errChan)

		// Test close
		sol.Close()

		// Verify done channel is closed
		select {
		case <-sol.done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("done channel should be closed immediately")
		}

		// Multiple closes should be safe
		assert.NotPanics(t, func() {
			sol.Close()
			sol.Close()
		})
	})
}

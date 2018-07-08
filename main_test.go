package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	mockhttp "github.com/karupanerura/go-mock-http-response"
)

func mockResponse(statusCode int, headers map[string]string, body []byte) {
	http.DefaultClient = mockhttp.NewResponseMock(statusCode, headers, body).MakeClient()
}

func TestRoundUp(t *testing.T) {
	//t.Parallel()
	testCases := []struct {
		name string
		in   int64
		out  int64
	}{
		{"roundup 99", 99, 1},
		{"roundup 1", 1, 99},
		{"roundup 0", 0, 0},
		{"roundup 100", 100, 0},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			total := roundUp(tc.in)
			if total != tc.out {
				t.Errorf("%v failed, got: %d, want: %d.", tc.name, total, tc.out)
			}
		})
	}
}

func TestValidateSignature(t *testing.T) {
	//t.Parallel()
	// Discard logging info
	log.SetOutput(ioutil.Discard)
	body := []byte(`{"test":"body"}`)
	signature := "C3zcs4qlrazPXGdPacksD/RhFeqBIjm/YkOjvZPo28OxJaUgaZT3RoTuJyGmlJkDWz/viPyWJvTJLbRz2tE7ww=="
	testCases := []struct {
		name string
		body []byte
		sig  string
		res  bool
	}{
		{"valid signature and body", body, signature, true},
		{"invalid signature", body, "foobar", false},
		{"invalid body", []byte(`{"foo":"bar"}`), signature, false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			if validateSignature(tc.body, tc.sig) != tc.res {
				t.Errorf("%v failed", tc.name)
			}
		})
	}
}

func TestTxnHandler(t *testing.T) {
	//t.Parallel()
	body := `{"test":"body"}`
	signature := "C3zcs4qlrazPXGdPacksD/RhFeqBIjm/YkOjvZPo28OxJaUgaZT3RoTuJyGmlJkDWz/viPyWJvTJLbRz2tE7ww=="
	testCases := []struct {
		name      string
		method    string
		headerSig string
		body      string
		message   string
		mockresp  []byte
	}{
		{"empty GET", http.MethodGet, "", "", "INFO: empty body", []byte{}},
		{"empty POST", http.MethodPost, "", "", "INFO: empty body", []byte{}},
		{"invalid signature", http.MethodPost, "", body, "ERROR: invalid request signature received", []byte{}},
		{"valid signature", http.MethodPost, signature, body, "", []byte{}},
		{
			"invalid json",
			http.MethodPost,
			"gKVP/neQpjsGl+nGYx4SmXtlNalLzrEmNaV03B353DN99S7hw40RQZ6c5l9puqnohJUjfu458HKPF4EzxVyW4w==",
			`{"foo":"bar}`,
			"ERROR: failed to unmarshal web hook payload",
			[]byte{},
		},
		{
			"non-card: outgoing transaction",
			http.MethodPost,
			"QTw8g8mjiOTLbJDZJZgFgVDa/SGaRglG2eUcSES7x/R0/MPxlCpbt3clmf/prcWrgL/IXJfgS9BDvrfgMn/AkA==",
			`{"content":{"type":"DIRECT_DEBIT"}}`,
			"INFO: ignoring DIRECT_DEBIT transaction",
			[]byte{},
		},
		{
			"card: outbound transaction",
			http.MethodPost,
			"XZCz9+Bx2RoaGL+0VFG1Gc/4cGpzQTHBcL+Rgh+LySuehkXZmCBnbquXE17/pDMx4l4JprdtlzOM3I3renRAFw==",
			`{"content":{"type":"TRANSACTION_CARD","amount": -24.99}}`,
			"INFO: round-up yields",
			[]byte{},
		},
		{
			"mobile wallet: outbound transaction",
			http.MethodPost,
			"7vA/GL44+7nfCRZWL4hy0AcakKEQvRmoJi0KmxO0ZhrqvndC0jSrzY0/LH5SjeR6qCZdZB3Jlhms5T7hWh51zg==",
			`{"content":{"type":"TRANSACTION_MOBILE_WALLET","amount": -24.99}}`,
			"INFO: round-up yields",
			[]byte{},
		},
		{
			"card: nothing to roundup",
			http.MethodPost,
			"naYnA204dwEn54SLx0Y2sGJDWOdoVfg4SSdLMwdQElNhRaoC+W2krSy6YWxwV6RwfI0zj439VTdzwoZy8rkhTw==",
			`{"content":{"type":"TRANSACTION_CARD","amount": -1.00}}`,
			"INFO: nothing to round-up",
			[]byte{},
		},
		{
			"mobile wallet: nothing to roundup",
			http.MethodPost,
			"oZC2cATjh3vAi5gLUd05/4lHuhP4GYcYLCAUHdB4Of0DJWyCfNsCGlTONuuKskkHK6E4/Zs+fqIkHVHzPNXKaQ==",
			`{"content":{"type":"TRANSACTION_MOBILE_WALLET","amount": -1.00}}`,
			"INFO: nothing to round-up",
			[]byte{},
		},
		{
			"card: outbound transaction",
			http.MethodPost,
			"XZCz9+Bx2RoaGL+0VFG1Gc/4cGpzQTHBcL+Rgh+LySuehkXZmCBnbquXE17/pDMx4l4JprdtlzOM3I3renRAFw==",
			`{"content":{"type":"TRANSACTION_CARD","amount": -24.99}}`,
			"INFO: round-up successful",
			[]byte{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			// Use a faux logger so we can parse the content to find our debug messages to confirm our tests
			// Set a mock response, if needed.
			if tc.mockresp != nil {
				mockResponse(http.StatusOK, map[string]string{"Content-Type": "application/json"}, tc.mockresp)
			}

			var fauxLog bytes.Buffer
			log.SetOutput(&fauxLog)
			req := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.body))
			req.Header.Add("X-Hook-Signature", tc.headerSig)
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(TxnHandler)
			handler.ServeHTTP(rr, req)
			if !strings.Contains(fauxLog.String(), tc.message) {
				t.Errorf("'%v' failed.\nGot:\n%v\nExpected:\n%v", tc.name, fauxLog.String(), tc.message)
			}
		})
	}
}

//var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//  w.WriteHeader(http.StatusOK)
//}))

// go test -coverprofile=c.out && go tool cover -html=c.out -o coverage.html
// open coverage.html

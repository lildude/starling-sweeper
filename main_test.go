package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
	// Discard logging info
	log.SetOutput(ioutil.Discard)
	body := `{"test":"body"}`
	signature := "C3zcs4qlrazPXGdPacksD/RhFeqBIjm/YkOjvZPo28OxJaUgaZT3RoTuJyGmlJkDWz/viPyWJvTJLbRz2tE7ww=="
	testCases := []struct {
		name       string
		method     string
		headerSig  string
		body       string
		statusCode int
	}{
		{"empty GET", http.MethodGet, "", "", http.StatusOK},
		{"empty POST", http.MethodPost, "", "", http.StatusOK},
		{"invalid signature", http.MethodPost, "", body, http.StatusBadRequest},
		{"valid signature", http.MethodPost, signature, body, http.StatusOK},
		{
			"invalid json",
			http.MethodPost,
			"gKVP/neQpjsGl+nGYx4SmXtlNalLzrEmNaV03B353DN99S7hw40RQZ6c5l9puqnohJUjfu458HKPF4EzxVyW4w==",
			`{"foo":"bar}`,
			http.StatusBadRequest,
		},
		{
			"non-card transaction",
			http.MethodPost,
			"QTw8g8mjiOTLbJDZJZgFgVDa/SGaRglG2eUcSES7x/R0/MPxlCpbt3clmf/prcWrgL/IXJfgS9BDvrfgMn/AkA==",
			`{"content":{"type":"DIRECT_DEBIT"}}`,
			http.StatusOK,
		},
		{
			"inbound transaction",
			http.MethodPost,
			"K+xd4/3TnmpDU8rrCkpmD8rbmQwW4KPBS6KrhOtg8pgxiG5cHnv1HWLAUbJYaUFmUD9rdcyg+fnysXaBJ6sqWQ==",
			`{"content":{"type":"TRANSACTION_CARD","amount": 24.99}}`,
			http.StatusOK,
		},
		{
			"nothing to roundup",
			http.MethodPost,
			"naYnA204dwEn54SLx0Y2sGJDWOdoVfg4SSdLMwdQElNhRaoC+W2krSy6YWxwV6RwfI0zj439VTdzwoZy8rkhTw==",
			`{"content":{"type":"TRANSACTION_CARD","amount": -1.00}}`,
			http.StatusOK,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			req := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.body))
			req.Header.Add("X-Hook-Signature", tc.headerSig)
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(TxnHandler)
			handler.ServeHTTP(rr, req)
			if status := rr.Code; status != tc.statusCode {
				t.Errorf("%v failed, got %v, expected %v", tc.name, status, tc.statusCode)
			}
		})
	}
}

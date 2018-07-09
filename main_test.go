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

// TODO: Skip signature verification during these tests
func TestTxnHandler(t *testing.T) {
	//t.Parallel()
	testCases := []struct {
		name     string
		method   string
		body     string
		message  string
		mockresp []byte
	}{
		{"empty GET", http.MethodGet, "", "INFO: empty body", []byte{}},
		{"empty POST", http.MethodPost, "", "INFO: empty body", []byte{}},
		{
			"invalid json",
			http.MethodPost,
			`{"foo":"bar}`,
			"ERROR: failed to unmarshal web hook payload",
			[]byte{},
		},
		{
			"non-card outgoing transaction",
			http.MethodPost,
			`{"content":{"type":"DIRECT_DEBIT"}}`,
			"INFO: ignoring DIRECT_DEBIT transaction",
			[]byte{},
		},
		{
			"card outbound transaction",
			http.MethodPost,
			`{"content":{"type":"TRANSACTION_CARD","amount": -24.99}}`,
			"INFO: transfer successful",
			[]byte(`{"transferUid":"12345-67890","success":true,"errors":[]}`),
		},
		{
			"card inbound transaction",
			http.MethodPost,
			`{"content":{"type":"TRANSACTION_CARD","amount": 24.99}}`,
			"INFO: ignoring inbound TRANSACTION_CARD transaction",
			[]byte{},
		},
		{
			"card nothing to roundup",
			http.MethodPost,
			`{"content":{"type":"TRANSACTION_CARD","amount": -1.00}}`,
			"INFO: nothing to transfer",
			[]byte{},
		},
		{
			"mobile wallet outbound transaction",
			http.MethodPost,
			`{"content":{"type":"TRANSACTION_MOBILE_WALLET","amount": -24.99}}`,
			"INFO: transfer successful",
			[]byte(`{"transferUid":"12345-67890","success":true,"errors":[]}`),
		},
		{
			"mobile wallet inbound transaction",
			http.MethodPost,
			`{"content":{"type":"TRANSACTION_MOBILE_WALLET","amount": 24.99}}`,
			"INFO: ignoring inbound TRANSACTION_MOBILE_WALLET transaction",
			[]byte{},
		},
		{
			"mobile wallet nothing to roundup",
			http.MethodPost,
			`{"content":{"type":"TRANSACTION_MOBILE_WALLET","amount": -1.00}}`,
			"INFO: nothing to transfer",
			[]byte{},
		},
		{
			"non-card inbound above threshold",
			http.MethodPost,
			`{"content":{"type":"FASTER_PAYMENTS_IN","amount": 2500.00}}`,
			"INFO: transfer successful",
			[]byte(`{"amount": 2500.00, "balance": 2754.12}`),
		},
		{
			"non-card inbound below threshold",
			http.MethodPost,
			`{"content":{"type":"FASTER_PAYMENTS_IN","amount": 500.00}}`,
			"INFO: ignoring inbound transaction below sweep threshold",
			[]byte(`{"amount": 500.00, "balance": 754.12}`),
		},
		{
			"card duplicate webhook delivery 1",
			http.MethodPost,
			`{"content":{"type":"TRANSACTION_CARD","amount": -24.99,"transactionUid":"test-trans-uid"}}`,
			"INFO: transfer successful",
			[]byte{},
		},
		{
			"card duplicate webhook delivery 2",
			http.MethodPost,
			`{"content":{"type":"TRANSACTION_CARD","amount": -24.99,"transactionUid":"test-trans-uid"}}`,
			"INFO: ignoring duplicate webhook delivery",
			[]byte{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			// Skip signature verification
			os.Setenv("SKIP_SIG", "1")
			s.SweepThreshold = 1000.00
			// Set a mock response, if needed.
			if len(tc.mockresp) > 0 {
				mockResponse(http.StatusOK, map[string]string{"Content-Type": "application/json"}, tc.mockresp)
			}
			// Use a faux logger so we can parse the content to find our debug messages to confirm our tests
			var fauxLog bytes.Buffer
			log.SetOutput(&fauxLog)
			req := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.body))
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

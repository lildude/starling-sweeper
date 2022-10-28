package feeditem

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	mockhttp "github.com/karupanerura/go-mock-http-response"
)

func mockResponse(statusCode int, headers map[string]string, body []byte) {
	http.DefaultClient = mockhttp.NewResponseMock(statusCode, headers, body).MakeClient()
}

func TestHandler(t *testing.T) {
	t.Parallel()
	r := miniredis.RunT(t)
	t.Cleanup(r.Close)
	t.Setenv("REDIS_URL", fmt.Sprintf("redis://%s", r.Addr()))

	testCases := []struct {
		name      string
		method    string
		body      string
		goal      string
		message   string
		mockresp  []byte
		signature string
	}{
		{
			name:      "card inbound transaction",
			method:    http.MethodPost,
			body:      `{"content":{"amount": {"minorUnits":2499},"source":"MASTER_CARD","direction":"IN"}}`,
			goal:      "sweep",
			message:   "[INFO] ignoring MASTER_CARD transaction",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "inbound above threshold",
			method:    http.MethodPost,
			body:      `{"content":{"amount": {"minorUnits": 250000},"source":"FASTER_PAYMENTS_IN","direction":"IN"}}`,
			goal:      "sweep",
			message:   "[INFO] transfer successful (Txn:  | 254.12)",
			mockresp:  []byte(`{"effectiveBalance": {"currency": "GBP",	"minorUnits": 275412}}`),
			signature: "",
		},
		{
			name:      "inbound below threshold",
			method:    http.MethodPost,
			body:      `{"content":{"amount": {"minorUnits":50000},"source":"FASTER_PAYMENTS_IN","direction":"IN"}}`,
			goal:      "sweep",
			message:   "[INFO] ignoring inbound transaction below sweep threshold",
			mockresp:  []byte(`{"amount": 500.00, "balance": 754.12}`),
			signature: "",
		},
		{
			name:      "overdrawn",
			method:    http.MethodPost,
			body:      `{"content":{"amount": {"minorUnits": 250000},"source":"FASTER_PAYMENTS_IN","direction":"IN"}}`,
			goal:      "sweep",
			message:   "[INFO] nothing to transfer",
			mockresp:  []byte(`{"effectiveBalance": {"currency": "GBP",	"minorUnits": -275412}}`),
			signature: "",
		},
		{
			name:      "no sweep goal set",
			method:    http.MethodPost,
			body:      `{"content":{"amount": {"minorUnits": 250000},"source":"FASTER_PAYMENTS_IN","direction":"IN"}}`,
			goal:      "",
			message:   "[INFO] no sweep savings goal set. Nothing to do.",
			mockresp:  []byte(`{"effectiveBalance": {"currency": "GBP",	"minorUnits": 275412}}`),
			signature: "",
		},
		{
			name:      "duplicate webhook",
			method:    http.MethodPost,
			body:      `{"webhookEventUid":"test-trans-uid","content":{"amount":{"minorUnits": 2499},"source":"MASTER_CARD","direction":"OUT"}}`,
			goal:      "sweep",
			message:   "[INFO] ignoring duplicate webhook delivery",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "bad signature",
			method:    http.MethodPost,
			body:      `{"webhookEventUid":"test-trans-uid","content":{"amount":{"minorUnits": 2499},"source":"MASTER_CARD","direction":"OUT"}}`,
			goal:      "sweep",
			message:   "[ERROR]",
			mockresp:  []byte{},
			signature: "12345",
		},
		{
			name:      "forced failure to get balance",
			method:    http.MethodPost,
			body:      `{"content":{"amount": {"minorUnits": 250000},"source":"FASTER_PAYMENTS_IN","direction":"IN"}}`,
			goal:      "sweep",
			message:   "[ERROR] problem getting balance",
			mockresp:  []byte(`{"broken": "json`),
			signature: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Skip signature verification
			if tc.signature == "" {
				os.Setenv("SKIP_SIG", "1")
			} else {
				os.Unsetenv("SKIP_SIG")
			}

			t.Setenv("SWEEP_GOAL", tc.goal)
			t.Setenv("SWEEP_THRESHOLD", "100000")

			// Set a mock response, if needed.
			if len(tc.mockresp) > 0 {
				mockResponse(http.StatusOK, map[string]string{"Content-Type": "application/json"}, tc.mockresp)
			}

			// Set Redis key if duplicate test
			if tc.name == "duplicate webhook" {
				_ = r.Set("starling_webhookevent_uid", "test-trans-uid")
			}
			// Use a faux logger so we can parse the content to find our debug messages to confirm our tests
			var fauxLog bytes.Buffer
			log.SetOutput(&fauxLog)
			req := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(Handler)
			handler.ServeHTTP(rr, req)
			if !strings.Contains(fauxLog.String(), tc.message) {
				t.Errorf("'%v' failed.\nGot:\n%v\nExpected:\n%v", tc.name, fauxLog.String(), tc.message)
			}
		})
	}
}

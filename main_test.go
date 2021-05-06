package main

import (
	"bytes"
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
	body := []byte(`{"one":"Value","two":"Other"}`)
	s.PublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAgIdCVYnz6JOFT7GGtjrMg4uaPRGGs5VlglSSd9i2i73zRp7AwZm8O/3LM5kPuPONOysJpdVSz9x6VGsRcaKkvMaOfYWYa6fe4l5IFiM8Z+WaL0WjIebdJOOjWxH3q/kW6KclwKBW0+2iNZPcZocllCOjPn/swp2MdhKLJOQkdB/1Q8Emxr6tsOlJkc2lWpXdtPHWUbBp31eF5/eDmuVCCBhTL76UyogQNgRV5qH2g/a2bNcNgTThR0PntXJLy2HLi9cEfXepevpoJM8HXNdaFwZV4pQUEzm3/jG7zI3isXnvtffG4uTIR8Q35yDrYeN8pX+zOAcnJYNbr9xdFEv7JQIDAQAB"
	signature := "KDGgtd7VDeyvNdyafyXNVZM8l/0zohWze5UCt1N0mbzCZ1f23nYEgnLrFvTRYADnToat/axKOGeXjiOBWJh/FcPvcWParx8x5d35j2u76/UmRPKjo8jxtMspmN27WlPdtTRr9kqHdDHUg80/9z1qKuEcUfm4EQX52NOvozDMb4qyYorgxaFCwUwMdZNskArIBTeJBtULAOtJqnEGipKRtRjeU6j2xD2uNzc3Vcy3+tdImRfqbX6SkS44zgkcFua6xEc09qRnRvLd+bxjSIufQ/wU695Uej9AtFg7MlrRCUaEZ2SVkNcmOUdRP2q882Y9mWGDIXdk66QHCVfCVu7pog=="

	err := validateSignature(body, signature)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
}

func TestValidateSignatureInvalid(t *testing.T) {
	//t.Parallel()
	s.PublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAgIdCVYnz6JOFT7GGtjrMg4uaPRGGs5VlglSSd9i2i73zRp7AwZm8O/3LM5kPuPONOysJpdVSz9x6VGsRcaKkvMaOfYWYa6fe4l5IFiM8Z+WaL0WjIebdJOOjWxH3q/kW6KclwKBW0+2iNZPcZocllCOjPn/swp2MdhKLJOQkdB/1Q8Emxr6tsOlJkc2lWpXdtPHWUbBp31eF5/eDmuVCCBhTL76UyogQNgRV5qH2g/a2bNcNgTThR0PntXJLy2HLi9cEfXepevpoJM8HXNdaFwZV4pQUEzm3/jG7zI3isXnvtffG4uTIR8Q35yDrYeN8pX+zOAcnJYNbr9xdFEv7JQIDAQAB"
	testCases := []struct {
		name string
		body []byte
		sig  string
	}{
		{
			name: "empty body",
			body: []byte(``),
			sig: "KDGgtd7VDeyvNdyafyXNVZM8l/0zohWze5UCt1N0mbzCZ1f23nYEgnLrFvTRYADnToat/axKOGeXjiOBWJh/FcPvcWParx8x5d35j2u76/UmRPKjo8jxtMspmN27WlPdtTRr9kqHdDHUg80/9z1qKuEcUfm4EQX52NOvozDMb4qyYorgxaFCwUwMdZNskArIBTeJBtULAOtJqnEGipKRtRjeU6j2xD2uNzc3Vcy3+tdImRfqbX6SkS44zgkcFua6xEc09qRnRvLd+bxjSIufQ/wU695Uej9AtFg7MlrRCUaEZ2SVkNcmOUdRP2q882Y9mWGDIXdk66QHCVfCVu7pog==",
		},
		{
			name: "invalid signature",
			body: []byte(`{"one":"Value","two":"Other"}`),
			sig:  "foobar",
		},
		{
			name: "invalid body",
			body: []byte(`{"foo":"bar"}`),
			sig:  "KDGgtd7VDeyvNdyafyXNVZM8l/0zohWze5UCt1N0mbzCZ1f23nYEgnLrFvTRYADnToat/axKOGeXjiOBWJh/FcPvcWParx8x5d35j2u76/UmRPKjo8jxtMspmN27WlPdtTRr9kqHdDHUg80/9z1qKuEcUfm4EQX52NOvozDMb4qyYorgxaFCwUwMdZNskArIBTeJBtULAOtJqnEGipKRtRjeU6j2xD2uNzc3Vcy3+tdImRfqbX6SkS44zgkcFua6xEc09qRnRvLd+bxjSIufQ/wU695Uej9AtFg7MlrRCUaEZ2SVkNcmOUdRP2q882Y9mWGDIXdk66QHCVfCVu7pog==",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			err := validateSignature(tc.body, tc.sig); if err == nil {
				t.Errorf("Expected error for: %v but didn't get one", tc.name)
			}
		})
	}
}

func TestTxnHandler(t *testing.T) {
	//t.Parallel()
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
			name:      "empty GET",
			method:    http.MethodGet,
			body:      "",
			goal:      "round",
			message:   "INFO: empty body",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "empty POST",
			method:    http.MethodPost,
			body:      "",
			goal:      "round",
			message:   "INFO: empty body",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "invalid json",
			method:    http.MethodPost,
			body:      `{"foo":"bar}`,
			goal:      "round",
			message:   "ERROR: failed to unmarshal web hook payload",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "non-card outgoing transaction",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_DIRECT_DEBIT"}`,
			goal:      "round",
			message:   "INFO: ignoring TRANSACTION_DIRECT_DEBIT transaction",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "card outbound transaction",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_CARD","content":{"amount": -24.99}}`,
			goal:      "round",
			message:   "INFO: transfer successful (Txn: 12345-67890 | 0.01)",
			mockresp:  []byte(`{"transferUid":"12345-67890","success":true,"errors":[]}`),
			signature: "",
		},
		{
			name:      "no roundup goal set",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_CARD","content":{"amount": -24.99}}`,
			goal:      "",
			message:   "INFO: no roundup savings goal set. Nothing to do.",
			mockresp:  []byte(`{"transferUid":"12345-67890","success":true,"errors":[]}`),
			signature: "",
		},
		{
			name:      "card inbound transaction",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_CARD","content":{"amount": 24.99}}`,
			goal:      "round",
			message:   "INFO: ignoring inbound TRANSACTION_CARD transaction",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "card nothing to roundup",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_CARD","content":{"amount": -1.00}}`,
			goal:      "round",
			message:   "INFO: nothing to transfer",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "mobile wallet outbound transaction",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_MOBILE_WALLET","content":{"amount": -24.99}}`,
			goal:      "round",
			message:   "INFO: transfer successful (Txn: 12345-67890 | 0.01)",
			mockresp:  []byte(`{"transferUid":"12345-67890","success":true,"errors":[]}`),
			signature: "",
		},
		{
			name:      "mobile wallet inbound transaction",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_MOBILE_WALLET","content":{"amount": 24.99}}`,
			goal:      "round",
			message:   "INFO: ignoring inbound TRANSACTION_MOBILE_WALLET transaction",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "mobile wallet nothing to roundup",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_MOBILE_WALLET","content":{"amount": -1.00}}`,
			goal:      "round",
			message:   "INFO: nothing to transfer",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "non-card inbound above threshold",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_FASTER_PAYMENT_IN","content":{"amount": 2500.00}}`,
			goal:      "sweep",
			message:   "INFO: transfer successful (Txn:  | 254.12)",
			mockresp:  []byte(`{"effectiveBalance": {"currency": "GBP",	"minorUnits": 275412}}`),
			signature: "",
		},
		{
			name:      "no sweep goal set",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_FASTER_PAYMENT_IN","content":{"amount": 2500.00}}`,
			goal:      "",
			message:   "INFO: no sweep savings goal set. Nothing to do.",
			mockresp:  []byte(`{"effectiveBalance": {"currency": "GBP",	"minorUnits": 275412}}`),
			signature: "",
		},
		{
			name:      "non-card inbound below threshold",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_FASTER_PAYMENT_IN","content":{"amount": 500.00}}`,
			goal:      "sweep",
			message:   "INFO: ignoring inbound transaction below sweep threshold",
			mockresp:  []byte(`{"amount": 500.00, "balance": 754.12}`),
			signature: "",
		},
		{
			name:      "card duplicate webhook delivery 1",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_CARD","webhookNotificationUid":"test-trans-uid","content":{"amount": -24.99}}`,
			goal:      "round",
			message:   "INFO: transfer successful",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "card duplicate webhook delivery 2",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_CARD","webhookNotificationUid":"test-trans-uid","content":{"amount": -24.99}}`,
			goal:      "round",
			message:   "INFO: ignoring duplicate webhook delivery",
			mockresp:  []byte{},
			signature: "",
		},
		{
			name:      "bad signature",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_CARD","webhookNotificationUid":"test-trans-uid","content":{"amount": -24.99}}`,
			goal:      "round",
			message:   "ERROR: invalid request signature receive",
			mockresp:  []byte{},
			signature: "12345",
		},
		{
			name:      "forced failure to get balance",
			method:    http.MethodPost,
			body:      `{"webhookType":"TRANSACTION_FASTER_PAYMENT_IN","content":{"amount": 2500.00}}`,
			goal:      "sweep",
			message:   "ERROR: problem getting balance",
			mockresp:  []byte(`{"broken": "json`),
			signature: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			// Skip signature verification
			if tc.signature == "" {
				os.Setenv("SKIP_SIG", "1")
			} else {
				os.Unsetenv("SKIP_SIG")
			}
			s.SavingGoal = ""
			s.SweepSavingGoal = ""
			if tc.goal == "round" {
				s.SavingGoal = "round"
			}
			if tc.goal == "sweep" {
				s.SweepSavingGoal = "sweep"
				s.SweepThreshold = 1000.00
			}
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

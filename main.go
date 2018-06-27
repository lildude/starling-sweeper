package main

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"

	"github.com/billglover/starling"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2"
)

type Specification struct {
	Port                string `required:"true" envconfig:"PORT"`
	WebhookSecret       string `required:"true" split_words:"true"`
	SavingGoal          string `required:"true" split_words:"true"`
	PersonalAccessToken string `required:"true" split_words:"true"`
}

var s Specification

func main() {
	log.SetFlags(0)
	err := envconfig.Process("starling", &s)
	if err != nil {
		log.Fatal(err.Error())
	}

	http.HandleFunc("/", TxnHandler)
	http.ListenAndServe(":"+s.Port, nil)
}

func TxnHandler(w http.ResponseWriter, r *http.Request) {
	// Grab body early as we'll need it later
	body, _ := ioutil.ReadAll(r.Body)
	if string(body) == "" {
		log.Println("INFO: empty body, pretending all is OK")
		w.WriteHeader(http.StatusOK)
		return
	}

	if !validateSignature(body, r.Header.Get("X-Hook-Signature")) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// Parse the contents of web hook payload and log pertinent items for debugging purposes
	wh := new(starling.WebHookPayload)
	err := json.Unmarshal([]byte(body), &wh)
	if err != nil {
		log.Println("ERROR: failed to unmarshal web hook payload:", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	log.Println("INFO: type:", wh.Content.Type)
	log.Println("INFO: amount:", wh.Content.Amount)

	// Don't round-up anything other than card transactions
	if wh.Content.Type != "TRANSACTION_CARD" && wh.Content.Type != "TRANSACTION_MOBILE_WALLET" {
		log.Println("INFO: ignoring non-card transaction")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Don't round-up incoming (i.e. positive) amounts
	if wh.Content.Amount >= 0.0 {
		log.Println("INFO: ignoring inbound transaction")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Round up to the nearest major unit
	amtMinor := math.Round(wh.Content.Amount * -100)
	ra := roundUp(int64(amtMinor))
	log.Println("INFO: round-up yields:", ra)

	// Don't try and transfer a zero value to the savings goal
	if ra == 0 {
		log.Println("INFO: nothing to round-up")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Transfer the funds to the savings goal
	ctx := context.Background()
	sb := newClient(ctx, s.PersonalAccessToken)
	amt := starling.Amount{
		MinorUnits: ra,
		Currency:   wh.Content.SourceCurrency,
	}

	txn, resp, err := sb.AddMoney(ctx, s.SavingGoal, amt)
	if err != nil {
		log.Println("ERROR: failed to move money to savings goal:", err)
		log.Println("ERROR: Starling Bank API returned:", resp.Status)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("INFO: round-up successful:", txn)
	w.WriteHeader(http.StatusCreated)
	return
}

func newClient(ctx context.Context, token string) *starling.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	baseURL, _ := url.Parse(starling.ProdURL)
	opts := starling.ClientOptions{BaseURL: baseURL}
	return starling.NewClientWithOptions(tc, opts)
}

// Calculate the request signature and reject the request if it doesn't match the signature header
func validateSignature(body []byte, reqSig string) bool {

	// Allow skipping verification - only use during testing
	_, skip_sig := os.LookupEnv("SKIP_SIG")
	if skip_sig {
		log.Println("INFO: skipping signature verification")
		return true
	}

	sha512 := sha512.New()
	sha512.Write([]byte(s.WebhookSecret + string(body)))
	recSig := base64.StdEncoding.EncodeToString(sha512.Sum(nil))
	if reqSig != recSig {
		log.Println("WARN: reqSig", reqSig)
		log.Println("WARN: recSig", recSig)
		log.Println("WARN: invalid request signature received")
		return false
	}
	return true
}

func roundUp(txn int64) int64 {
	// By using 99 we ensure that a 0 value rounds is not rounded up
	// to the next 100.
	amtRound := (txn + 99) / 100 * 100
	return amtRound - txn
}

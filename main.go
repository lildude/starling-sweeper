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
	Port                string  `required:"true" envconfig:"PORT"`
	WebhookSecret       string  `required:"true" split_words:"true"`
	SavingGoal          string  `required:"true" split_words:"true"`
	PersonalAccessToken string  `required:"true" split_words:"true"`
	SweepThreshold      float64 `split_words:"true"`
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

// TODO: Add sweeper support - when a payment comes in above a set threshold, and we have a balance above 0, move the balance to a pot.
func TxnHandler(w http.ResponseWriter, r *http.Request) {
	// Return OK as soon as we've received the payload - the webhook doesn't care what we do with the payload so no point holding things back.
	w.WriteHeader(http.StatusOK)

	// Grab body early as we'll need it later
	body, _ := ioutil.ReadAll(r.Body)
	if string(body) == "" {
		log.Println("INFO: empty body, pretending all is OK")
		return
	}

	if !validateSignature(body, r.Header.Get("X-Hook-Signature")) {
		return
	}

	// Parse the contents of web hook payload and log pertinent items for debugging purposes
	wh := new(starling.WebHookPayload)
	err := json.Unmarshal([]byte(body), &wh)
	if err != nil {
		log.Println("ERROR: failed to unmarshal web hook payload:", err)
		return
	}
	log.Println("INFO: type:", wh.Content.Type)
	log.Println("INFO: amount:", wh.Content.Amount)

	// Ignore anything other than card transactions or transactions above the threshold if a threshold is set
	if (wh.Content.Type != "TRANSACTION_CARD" && wh.Content.Type != "TRANSACTION_MOBILE_WALLET") || (wh.Content.Amount > s.SweepThreshold) {
		log.Printf("INFO: ignoring %s transaction\n", wh.Content.Type)
		return
	}

	var ra int64

	if wh.Content.Type == "TRANSACTION_CARD" || wh.Content.Type == "TRANSACTION_MOBILE_WALLET" {
		if wh.Content.Amount >= 0.0 {
			log.Println("INFO: ignoring inbound card transaction")
			return
		}
		// Round up to the nearest major unit
		amtMinor := math.Round(wh.Content.Amount * -100)
		ra = roundUp(int64(amtMinor))
		log.Println("INFO: round-up yields:", ra)

		// Don't try and transfer a zero value to the savings goal
		if ra == 0 {
			log.Println("INFO: nothing to round-up")
			return
		}
	}

	if wh.Content.Amount > s.SweepThreshold {
		ra = getBalance(wh.Content.TransactionUID)
	}

	// Transfer the funds to the savings goal
	// TODO: Go noob goal: Learn how to stub the requests to the API to return OK
	//       Until then, we skip this section entirely if the access token env var isn't set,
	//       which will be the case during testing. We wouldn't get this far outside of testing.
	if s.PersonalAccessToken != "" {
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
			return
		}
		log.Println("INFO: Txn ID ", txn)
	}

	log.Println("INFO: round-up successful")
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
		log.Println("ERROR: invalid request signature received")
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

// Grabs txn deets and removes txn amt from balance and returns the minor units
func getBalance(txnUid string) int64 {
	return 0
	//ctx := context.Background()
	//sb := newClient(ctx, s.PersonalAccessToken)

}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/lildude/starling"
	"golang.org/x/oauth2"
)

// Settings pulled in from the environment variables.
// SavingGoal is now optional as Starling now does rounding itself, however the Starling API doesn't provide a way to determine this rounding yet.
type Settings struct {
	Port                string `required:"true" envconfig:"PORT"`
	WebhookSecret       string `required:"true" split_words:"true"`
	SavingGoal          string `split_words:"true"`
	PersonalAccessToken string `required:"true" split_words:"true"`
	SweepThreshold      int64  `split_words:"true"`
	SweepSavingGoal     string `split_words:"true"`
	AccountUID          string `required:"true" split_words:"true"`
	PublicKey           string `required:"true" split_words:"true"`
}

var s Settings

func main() {
	log.SetFlags(0)
	err := envconfig.Process("starling", &s)
	if err != nil {
		log.Fatal(err.Error())
	}
	// If s.SavingGoal is not set, we don't do rounding
	// If s.SweepSavingGoal is not set, we don't do sweeping
	// No point continuing if neither are set.
	if s.SweepSavingGoal == "" && s.SavingGoal == "" {
		log.Fatal("No savings goal set.")
	}

	http.HandleFunc("/feed-item", TxnHandler)
	fmt.Println("Starting server on port", s.Port)
	if err := http.ListenAndServe(":"+s.Port, nil); err != nil {
		log.Fatal(err.Error())
	}
}

// TxnHandler handles the incoming webhook event
func TxnHandler(w http.ResponseWriter, r *http.Request) {
	// Return OK as soon as we've received the payload - the webhook doesn't care what we do with the payload so no point holding things back.
	w.WriteHeader(http.StatusOK)

	// Grab body early as we'll need it later
	body, _ := ioutil.ReadAll(r.Body)
	if string(body) == "" {
		log.Println("INFO: empty body, pretending all is OK")
		return
	}
		// Debug
	log.Println("DEBUG sig:", r.Header.Get("X-Hook-Signature"))
	log.Println("DEBUG body:", string(body))
	// Allow skipping verification - only use during testing
	_, skipSig := os.LookupEnv("SKIP_SIG")
	if !skipSig {
		ok, err := starling.Validate(r, s.PublicKey)
		if !ok {
			log.Println("ERROR:", err)
			return
		}
	}

	// Parse the contents of web hook payload and log pertinent items for debugging purposes
	wh := new(starling.WebHookPayload)
	err := json.Unmarshal([]byte(body), &wh)
	if err != nil {
		log.Println("ERROR: failed to unmarshal web hook payload:", err)
		return
	}

	// Store the webhook uid in an environment variable and use to try catch duplicate deliveries
	ltu, _ := os.LookupEnv("LAST_TRANSACTION_UID")
	if ltu != "" && ltu == wh.WebhookEventUID {
		log.Println("INFO: ignoring duplicate webhook delivery")
		return
	}

	os.Setenv("LAST_TRANSACTION_UID", wh.WebhookEventUID)

	log.Printf("INFO: amount: %.2f", float64(wh.Content.Amount.MinorUnits/100))

	// Ignore anything other than card transactions or specific inbound transactions likely to be large payments like salary etc
	if wh.Content.Source != "MASTER_CARD" &&
		wh.Content.Source != "FASTER_PAYMENTS_IN" &&
		wh.Content.Source != "NOSTRO_DEPOSIT" &&
		wh.Content.Source != "DIRECT_CREDIT" {
		log.Printf("INFO: ignoring %s transaction\n", wh.Content.Source)
		return
	}

	var ru int64
	var destGoal string

	switch wh.Content.Source {
	case "MASTER_CARD":
		// Return early if no savings goal
		if s.SavingGoal == "" {
			log.Println("INFO: no roundup savings goal set. Nothing to do.")
			return
		}
		destGoal = s.SavingGoal
		if wh.Content.Direction == "IN" {
			log.Printf("INFO: ignoring inbound %s transaction\n", wh.Content.Source)
			return
		}
		// Round up to the nearest major unit
		ru = roundUp(int64(wh.Content.Amount.MinorUnits))
		log.Println("INFO: round-up yields:", ru)

	case "FASTER_PAYMENTS_IN", "NOSTRO_DEPOSIT", "DIRECT_CREDIT":
		// Return early if no savings goal
		if s.SweepSavingGoal == "" {
			log.Println("INFO: no sweep savings goal set. Nothing to do.")
			return
		}
		destGoal = s.SweepSavingGoal
		if s.SweepThreshold <= 0 || wh.Content.Amount.MinorUnits < s.SweepThreshold {
			log.Printf("INFO: ignoring inbound transaction below sweep threshold (%2.f)\n", float64(s.SweepThreshold/100))
			return
		}

		if wh.Content.Amount.MinorUnits > s.SweepThreshold {
			log.Printf("INFO: threshold: %.2f\n", float64(s.SweepThreshold/100))
			ru = getBalanceBefore(wh.Content.Amount.MinorUnits)
			log.Printf("INFO: balance before: %.2f\n", float64(ru)/100)
		}
	}

	// Don't try and transfer a zero value to the savings goal
	if ru == 0 {
		log.Println("INFO: nothing to transfer")
		return
	}

	ctx := context.Background()
	cl := newClient(ctx, s.PersonalAccessToken)
	amt := starling.Amount{
		MinorUnits: ru,
		Currency:   wh.Content.Amount.Currency,
	}

	// Transfer the funds to the savings goal
	txn, resp, err := cl.TransferToSavingsGoal(ctx, s.AccountUID, destGoal, amt)
	if err != nil {
		log.Println("ERROR: failed to move money to savings goal:", err)
		log.Println("ERROR: Starling Bank API returned:", resp.Status)
		return
	}

	log.Printf("INFO: transfer successful (Txn: %s | %.2f)", txn, float32(ru)/100)
}

func newClient(ctx context.Context, token string) *starling.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	baseURL, _ := url.Parse(starling.ProdURL)
	opts := starling.ClientOptions{BaseURL: baseURL}
	return starling.NewClientWithOptions(tc, opts)
}

func roundUp(txn int64) int64 {
	// By using 99 we ensure that a 0 value is not rounded up to the next 100.
	amtRound := (txn + 99) / 100 * 100
	return amtRound - txn
}

// Grabs txn deets and removes txn amt from balance and returns the minor units
func getBalanceBefore(txnAmt int64) int64 {
	ctx := context.Background()
	cl := newClient(ctx, s.PersonalAccessToken)
	bal, _, err := cl.AccountBalance(ctx, s.AccountUID)
	if err != nil {
		log.Println("ERROR: problem getting balance")
		return 0
	}
	log.Printf("INFO: balance: %.2f", float32(bal.Effective.MinorUnits)/100)
	diff := (bal.Effective.MinorUnits - txnAmt)
	return diff
}

package feeditem

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/lildude/starling"
	"github.com/lildude/starling-sweep/internal/cache"
	"golang.org/x/oauth2"
)

// Handler handles the incoming webhook event
func Handler(w http.ResponseWriter, r *http.Request) {
	// Return OK as soon as we've received the payload - the webhook doesn't care what we do with the payload so no point holding things back.
	w.WriteHeader(http.StatusOK)

	// Allow skipping verification - only use during testing
	_, skipSig := os.LookupEnv("SKIP_SIG")
	if !skipSig {
		ok, err := starling.Validate(r, os.Getenv("PUBLIC_KEY"))
		if !ok {
			log.Println("ERROR:", err)
			return
		}
	}

	// Parse the contents of web hook payload and log pertinent items for debugging purposes
	body, _ := io.ReadAll(r.Body)
	wh := new(starling.WebHookPayload)
	err := json.Unmarshal([]byte(body), &wh)
	if err != nil {
		log.Println("ERROR: failed to unmarshal web hook payload:", err)
		return
	}

	// Store the webhook uid in Redis and use to catch duplicate deliveries
	cache := cache.NewRedisCache(os.Getenv("REDIS_URL"))
	ltu, err := cache.Get("starling_webhookevent_uid")
	if err != nil {
		log.Println("ERROR: failed to get starling_webhookevent_uid from cache:", err)
		return
	}

	if ltu != "" && ltu == wh.WebhookEventUID {
		log.Println("INFO: ignoring duplicate webhook delivery")
		return
	}

	// Store the webhook uid in Redis for future reference
	err = cache.Set("starling_webhookevent_uid", wh.WebhookEventUID)
	if err != nil {
		log.Println("ERROR: failed to set starling_webhookevent_uid in cache:", err)
		return
	}

	log.Printf("INFO: amount: %.2f", float64(wh.Content.Amount.MinorUnits)/100)

	// Ignore anything other than specific inbound transactions likely to be large payments like salary etc
	if wh.Content.Source != "FASTER_PAYMENTS_IN" &&
		wh.Content.Source != "NOSTRO_DEPOSIT" &&
		wh.Content.Source != "DIRECT_CREDIT" {
		log.Printf("INFO: ignoring %s transaction\n", wh.Content.Source)
		return
	}

	var balance int64
	var goal string

	switch wh.Content.Source {
	case "FASTER_PAYMENTS_IN", "NOSTRO_DEPOSIT", "DIRECT_CREDIT":
		// Return early if no savings goal
		goal = os.Getenv("SWEEP_GOAL")
		if goal == "" {
			log.Println("INFO: no sweep savings goal set. Nothing to do.")
			return
		}

		threshold, _ := strconv.ParseInt(os.Getenv("SWEEP_THRESHOLD"), 10, 64)
		if threshold <= 0 || wh.Content.Amount.MinorUnits < threshold {
			log.Printf("INFO: ignoring inbound transaction below sweep threshold (%2.f)\n", float64(threshold/100))
			return
		}

		if wh.Content.Amount.MinorUnits > threshold {
			log.Printf("INFO: threshold: %.2f\n", float64(threshold/100))
			balance = getBalanceBefore(wh.Content.Amount.MinorUnits)
			log.Printf("INFO: balance before: %.2f\n", float64(balance)/100)
		}
	}

	// Don't try and transfer a zero value to the savings goal
	if balance == 0 {
		log.Println("INFO: nothing to transfer")
		return
	}

	ctx := context.Background()
	cl := newClient(ctx, os.Getenv("PERSONAL_ACCESS_TOKEN"))
	amt := starling.Amount{
		MinorUnits: balance,
		Currency:   wh.Content.Amount.Currency,
	}

	// Transfer the funds to the savings goal
	txn, resp, err := cl.TransferToSavingsGoal(ctx, os.Getenv("ACCOUNT_UID"), goal, amt)
	if err != nil {
		log.Println("ERROR: failed to move money to savings goal:", err)
		log.Println("ERROR: Starling Bank API returned:", resp.Status)
		return
	}

	log.Printf("INFO: transfer successful (Txn: %s | %.2f)", txn, float32(balance)/100)
}

func newClient(ctx context.Context, token string) *starling.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	baseURL, _ := url.Parse(starling.ProdURL)
	opts := starling.ClientOptions{BaseURL: baseURL}
	return starling.NewClientWithOptions(tc, opts)
}

// Grabs txn deets and removes txn amt from balance and returns the minor units
func getBalanceBefore(txnAmt int64) int64 {
	ctx := context.Background()
	cl := newClient(ctx, os.Getenv("PERSONAL_ACCESS_TOKEN"))
	bal, _, err := cl.AccountBalance(ctx, os.Getenv("ACCOUNT_UID"))
	if err != nil {
		log.Println("ERROR: problem getting balance")
		return 0
	}
	log.Printf("INFO: balance: %.2f", float32(bal.Effective.MinorUnits)/100)
	diff := (bal.Effective.MinorUnits - txnAmt)
	return diff
}

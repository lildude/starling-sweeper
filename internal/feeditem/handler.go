// Package feeditem implements the webhook handler for Starling webhooks.
package feeditem

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/lildude/starling"
	"github.com/lildude/starling-sweep/internal/cache"
	"golang.org/x/oauth2"
)

// Handler handles the incoming webhook event.
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Return OK as soon as we've received the payload - the webhook doesn't care what we do with the payload so no point holding things back.
	w.WriteHeader(http.StatusOK)

	// Allow skipping verification - only use during testing.
	_, skipSig := os.LookupEnv("SKIP_SIG")
	if !skipSig {
		ok, err := starling.Validate(r, os.Getenv("PUBLIC_KEY"))
		if !ok {
			slog.Error("signature validation failed", "error", err)
			return
		}
	}

	r.ParseForm() //nolint:gosec // We're not using the form data for anything other than testing.
	_, dryRun := r.Form["dry-run"]
	if !dryRun {
		_, dryRun = r.Form["dryrun"]
	}

	// Parse the contents of web hook payload and log pertinent items for debugging purposes
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	wh := new(starling.WebHookPayload)
	err := json.Unmarshal(body, &wh)
	if err != nil {
		slog.Error("failed to unmarshal web hook payload", "error", err)
		return
	}

	// Store the webhook uid in Redis and use to catch duplicate deliveries
	rcache, err := cache.NewRedisCache(ctx, os.Getenv("REDIS_URL"))
	if err != nil {
		slog.Error("unable to create redis cache", "error", err)
		return
	}
	ltu, err := rcache.Get(ctx, "starling_webhookevent_uid")
	if err != nil {
		slog.Error("failed to get starling_webhookevent_uid from cache", "error", err)
		return
	}

	if ltu != "" && ltu == wh.WebhookEventUID {
		slog.Info("ignoring duplicate webhook delivery")
		return
	}

	// Store the webhook uid in Redis for future reference
	err = rcache.Set(ctx, "starling_webhookevent_uid", wh.WebhookEventUID)
	if err != nil {
		slog.Error("failed to set starling_webhookevent_uid in cache", "error", err)
		return
	}

	slog.Info("received transaction", "amount", float64(wh.Content.Amount.MinorUnits)/100)

	// Ignore anything other than specific inbound transactions likely to be large payments like salary etc
	if wh.Content.Source != "FASTER_PAYMENTS_IN" &&
		wh.Content.Source != "NOSTRO_DEPOSIT" &&
		wh.Content.Source != "DIRECT_CREDIT" {
		slog.Info("ignoring transaction", "source", wh.Content.Source)
		return
	}

	var balance int64

	// Return early if no savings goal
	goal := os.Getenv("SWEEP_GOAL")
	if goal == "" {
		slog.Info("no sweep savings goal set. Nothing to do.")
		return
	}

	threshold, _ := strconv.ParseInt(os.Getenv("SWEEP_THRESHOLD"), 10, 64)
	if threshold <= 0 || wh.Content.Amount.MinorUnits < threshold {
		slog.Info("ignoring inbound transaction below sweep threshold", "threshold", float64(threshold)/100)
		return
	}

	if wh.Content.Amount.MinorUnits > threshold {
		slog.Info("threshold met", "threshold", float64(threshold)/100)
		balance, err = getBalanceBefore(ctx, wh.Content.Amount.MinorUnits)
		if err != nil {
			slog.Error("problem getting balance", "error", err)
			return
		}
		slog.Info("balance before", "amount", float64(balance)/100)
	}

	// Don't try and transfer a zero or overdrawn value to the savings goal
	if balance <= 0 {
		slog.Info("nothing to transfer")
		return
	}

	cl := newClient(ctx, os.Getenv("PERSONAL_ACCESS_TOKEN"))
	amt := starling.Amount{
		MinorUnits: balance,
		Currency:   wh.Content.Amount.Currency,
	}

	// Transfer the funds to the savings goal
	if dryRun {
		slog.Info("dry run: would transfer", "amount", float64(balance)/100)
	} else {
		_, resp, err := cl.TransferToSavingsGoal(ctx, os.Getenv("ACCOUNT_UID"), goal, amt)
		if err != nil {
			slog.Error("failed to move money to savings goal", "error", err)
			return
		}
		defer resp.Body.Close()
		slog.Info("transfer successful", "amount", float64(balance)/100)
	}
}

func newClient(ctx context.Context, token string) *starling.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	baseURL, _ := url.Parse(starling.ProdURL)
	opts := starling.ClientOptions{BaseURL: baseURL}
	return starling.NewClientWithOptions(tc, opts)
}

// getBalanceBefore grabs the current balance and subtracts the transaction amount.
func getBalanceBefore(ctx context.Context, txnAmt int64) (int64, error) {
	cl := newClient(ctx, os.Getenv("PERSONAL_ACCESS_TOKEN"))
	bal, resp, err := cl.AccountBalance(ctx, os.Getenv("ACCOUNT_UID"))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	diff := bal.Effective.MinorUnits - txnAmt

	return diff, nil
}

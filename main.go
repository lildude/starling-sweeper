package main

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"

	"github.com/billglover/starling"
	"golang.org/x/oauth2"
)

var secret string
var token string
var goal string

func main() {
	port := os.Getenv("PORT")
	secret := os.Getenv("STARLING_WEBHOOK_SECRET")
	goal := os.Getenv("STARLING_SAVING_GOAL")
	token := os.Getenv("STARLING_PERSONAL_ACCESS_TOKEN")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	if secret == "" {
		log.Fatal("$STARLING_WEBHOOK_SECRET must be set")
	}
	if goal == "" {
		log.Fatal("$STARLING_SAVING_GOAL must be set")
	}
	if token == "" {
		log.Fatal("STARLING_PERSONAL_ACCESS_TOKEN must be set")
	}

	http.HandleFunc("/", TxnHandler)
	http.ListenAndServe(":"+port, nil)
}

func TxnHandler(w http.ResponseWriter, r *http.Request) {

	// Calculate the request signature and reject the request if it doesn't match the signature header
	sha512 := sha512.New()
	sha512.Write([]byte(secret + request.Body))
	recSig := base64.StdEncoding.EncodeToString(sha512.Sum(nil))
	reqSig := request.Headers["X-Hook-Signature"]
	if reqSig != recSig {
		log.Println("WARN: invalid request signature received")
		return clientError(http.StatusBadRequest)
	}

	// Parse the contents of web hook payload and log pertinent items for debugging purposes
	wh := new(starling.WebHookPayload)
	err := json.Unmarshal([]byte(request.Body), &wh)
	if err != nil {
		log.Println("ERROR: failed to unmarshal web hook payload:", err)
		return serverError(err)
	}
	log.Println("INFO: type:", wh.Content.Type)
	log.Println("INFO: amount:", wh.Content.Amount)

	// Don't round-up anything other than card transactions
	if wh.Content.Type != "TRANSACTION_CARD" && wh.Content.Type != "TRANSACTION_MOBILE_WALLET" {
		log.Println("INFO: ignoring non-card transaction")
		return success()
	}

	// Don't round-up incoming (i.e. positive) amounts
	if wh.Content.Amount >= 0.0 {
		log.Println("INFO: ignoring inbound transaction")
		return success()
	}

	// Round up to the nearest major unit
	amtMinor := math.Round(wh.Content.Amount * -100)
	ra := roundUp(int64(amtMinor))
	log.Println("INFO: round-up yields:", ra)

	// Don't try and transfer a zero value to the savings goal
	if ra == 0 {
		log.Println("INFO: nothing to round-up")
		return success()
	}

	// Transfer the funds to the savings goal
	ctx := context.Background()
	sb := newClient(ctx, token)
	amt := starling.Amount{
		MinorUnits: ra,
		Currency:   wh.Content.SourceCurrency,
	}

	txn, resp, err := sb.AddMoney(ctx, goal, amt)
	if err != nil {
		log.Println("ERROR: failed to move money to savings goal:", err)
		log.Println("ERROR: Starling Bank API returned:", resp.Status)
		return serverError(err)
	}

	log.Println("INFO: round-up successful:", txn)
	return success()
}

func success() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "",
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

func clientError(status int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       http.StatusText(status),
	}, nil
}

func newClient(ctx context.Context, token string) *starling.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	baseURL, _ := url.Parse(starling.ProdURL)
	opts := starling.ClientOptions{BaseURL: baseURL}
	return starling.NewClientWithOptions(tc, opts)
}

func roundUp(txn int64) int64 {
	// By using 99 we ensure that a 0 value rounds is not rounded up
	// to the next 100.
	amtRound := (txn + 99) / 100 * 100
	return amtRound - txn

}

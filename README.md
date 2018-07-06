# starling-roundup

This application allows you to round-up your Starling bank transactions to the nearest £1 and transfer the delta to a savings goal.

This implementation is a fork of the original work at https://github.com/billglover/starling-roundup, but targeted at Heroku. Why Heroku? Because I already use Heroku, it has a simple "click" deploy method and gives me all the web server resources and functionality I need without having to string together, and individually pay for, a ton of AWS services. This runs quite happily in the free micro dyno.

## How it works

1. Starling Bank triggers a web-hook on each transaction.
2. This web-hook is configured to POST the transaction data to this application running on Heroku.
3. The application checks the signature of the request, checks the transaction UID and if it's not the transaction we rounded up, rounds up the value, and sends a request back to Starling Bank to move the delta to a savings goal.


## Questions

**How do you store secure parameters?** This application retrieves all parameters from the Heroku configuration variables, or the `.env` file when running locally, thus limiting access to just the application. The following three config vars are used:

- `STARLING_WEBHOOK_SECRET` - used to validate inbound requests
- `STARLING_PERSONAL_ACCESS_TOKEN` - used to request transfers to savings goal
- `STARLING_SAVING_GOAL` -  the identifier of the target savings goal

**Why don't you use a database?** This is still an early implementation, and I may do so in the future, but for the moment this is overkill for only storing a single value - the UID of last card transaction we processed.

## Installation

### Pre-Requisites

- A [Starling Bank](https://starlingbank.com) account
- A [Starling Bank Developer](https://developer.starlingbank.com) account
- A [Heroku](https://heroku.com) account

### Configuring Your App

- Deploy the application to Heroku: [Snazzy button coming :soon:]
- Take note of the application URL, this is the web-hook URL you'll need to enter on Starling.
- Register an application with your Starling developer account.
- Create a personal web-hook using the URL returned above.
- Make a note of the web-hook secret and the personal access token.
- Set the three config vars, named as above, either in the Heroku UI, or using the Heroku CLI:

```
$ heroku config:set STARLING_WEBHOOK_SECRET="your-secret" STARLING_PERSONAL_ACCESS_TOKEN="your-personal-access-token" STARLING_SAVING_GOAL="your-savings-goal-id"
```

### Local Development and Testing

- Save your Heroku config vars to a `.env` file: `heroku config:get -s  >.env`. Don't commit this file to your repo unless you really don't like your money.
- Start the application: `heroku local`.
- Send test requests to 0.0.0.0:5000 using something like curl or httpie.

### Contributing

Issues and pull requests are both welcome. I'd be particularly interested in help around packaging this up to simplify the deployment process.

## Similar Projects

- [Starling Roundup (for AWS)](https://github.com/billglover/starling-roundup) - the origin of this fork.
- [Starling CoinJar](https://github.com/cooperaj/starling-coinjar)

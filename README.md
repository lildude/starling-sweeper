# Starling Sweep

This application allows you to "sweep" the balance in your account as the time of receiving an inbound faster payment or Nostro deposit to a savings goal.

## How it works

1. Starling Bank triggers a webhook on each transaction.
1. This webhook is configured to POST the transaction data to this application running on Azure Functions.
1. The application...
   - checks the signature of the request,
   - checks the transaction UID and if it's not the transaction we swept last,
   - checks the amount is greater than a set threshold,
   - checks balance from prior to the incoming,
   - and then sends a request back to Starling Bank to move the original balance to a savings goal.

## Installation

### Pre-Requisites

- A [Starling Bank](https://starlingbank.com) account.
- A [Starling Bank Developer](https://developer.starlingbank.com) account.
- An [Azure](https://azure.microsoft.com/free/) account.
- A Redis database. I use a free account from [Redis](https://redis.com/try-free/) as it's cheaper than Azure.

### Configuring Your App

- Create your function app:
  - in the Azure portal:
    - Start by searching for Function App in the Azure Portal and click Create. The important settings for this are below, other settings you can use default or your own preferences.

    [Basic]
        Publish: Code
        Runtime stack: Custom Handler
        Version: custom

    [Hosting]
        Operating System: Linux
        Plan type: Consumption (Serverless)

  OR:
  
  - in [VSCode](https://learn.microsoft.com/en-us/azure/azure-functions/create-first-function-vs-code-other?tabs=go%2Clinux#create-the-function-app-in-azure)
- [Add credentials to the GitHub Actions secrets](https://github.com/Azure/functions-action#using-publish-profile-as-deployment-credential-recommended) for your repo if you plan to use GitHub Actions for deployment.
  - set `AZURE_FUNCTIONAPP_PUBLISH_PROFILE` with the content of the `.PublishSettings` file downloaded from the portal.
  - set `FUNCTION_ALL` to the name of your function.
- Register an application with your Starling developer account.
- Create a personal webhook using the URL from when you created your function app above.
- Make a note of the webhook secret and the personal access token.
- Set the following keys under Settings > Configuration > Application Settings for your function app:
  - `WEBHOOK_SECRET` - used to validate inbound requests.
  - `PERSONAL_ACCESS_TOKEN` - used to request transfers to savings goal.
  - `SWEEP_GOAL` - the target savings goal for sweeps.
  - `SWEEP_THRESHOLD` - the threshold, in _pence_, for incoming payments to trigger a sweep.
  - `ACCOUNT_UID` - the identifier of the Starling account on which you want this to run.
  - `REDIS_URL` - the URL for the Redis database you want to use.
  You should probably use the [Key Vault](https://azure.microsoft.com/services/key-vault/) for all secrets to be extra safe.
- Deploy the application, either using [VSCode](https://docs.microsoft.com/en-us/azure/azure-functions/create-first-function-vs-code-other?tabs=go%2Clinux#publish-the-project-to-azure) or via GitHub Actions by pushing to `main` or merging a pull request into `main`.

### Local Development and Testing

- [Configure your environment](https://learn.microsoft.com/en-us/azure/azure-functions/create-first-function-vs-code-other?tabs=go%2Cmacos#configure-your-environment)
- Save the above settings to a `.env` file.
  You can also add these to a `local.settings.json` file by [pulling them from Azure](https://learn.microsoft.com/en-us/azure/azure-functions/functions-develop-vs-code?tabs=csharp#download-settings-from-azure) and later push [push these to Azure when you deploy from VSCode](https://learn.microsoft.com/en-us/azure/azure-functions/functions-develop-vs-code?tabs=csharp#application-settings-in-azure).
  **Don't commit either of these files to your repo unless you really don't like your money.**
- Start the application: `make start`.

## Similar Projects

- [Starling Roundup (for AWS)](https://github.com/billglover/starling-roundup) - the origin of the fork of this fork.
- [Starling CoinJar](https://github.com/cooperaj/starling-coinjar)

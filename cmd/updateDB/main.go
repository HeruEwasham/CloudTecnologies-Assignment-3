package main

import "time"
import "net/http"
import "encoding/json"
import "github.com/HeruEwasham/CloudTecnologies-Assignment-3/exchange"
import "strconv"

func sendMessageCheckOK(msg exchange.MessageWebhook) {
	ok := exchange.SendMessageWebhook(msg)
	if !ok {
		println("Error when sending message-webhook")
	}
}

// Is called every day
func getCurrencyFromExternalDatabase(database exchange.Storage, date string, base string) (bool, exchange.Currency) { // Argument is here used for testing
	msg := exchange.MessageWebhook{}                               // Prepere errormessage if needed
	msg.Heading = "Couldn't save the latest currency to database!" // Prepare if error
	msg.DateTime = time.Now().Format("2006-01-02-15:04:05")
	msg.FromService = "Cloud tecnologies: Assignment 2"
	currency := exchange.Currency{}
	response, httpErr := http.Get("http://api.fixer.io/" + date + "?base=" + base) // Call api
	if httpErr != nil {
		err := "Failed to get fixer.io currency: " + httpErr.Error()
		msg.Message = err
		println(err)
		sendMessageCheckOK(msg)
		return false, currency
	}

	decodeErr := json.NewDecoder(response.Body).Decode(&currency)
	defer response.Body.Close()
	if decodeErr != nil {
		err := "Failed to get fixer.io currency decoded: " + httpErr.Error()
		msg.Message = err
		println(err)
		sendMessageCheckOK(msg)
		return false, currency
	}

	currencyAlreadyRegistered := false

	_, latestDate, _, err := database.GetLatest(base, "NOK")

	if err == nil { // We don't have an error, check that we havn't added it already
		if latestDate == currency.Date { // If the date is the same, we don't need to add it to our database
			currencyAlreadyRegistered = true
		}
	}

	if !currencyAlreadyRegistered {
		_, err = database.RegisterCurrencyToDatabase(currency)

		if err != nil {
			err := "Failed to register currency to database: " + err.Error()
			msg.Message = err
			println(err)
			sendMessageCheckOK(msg)
			return false, currency
		}

		webhooks, statusCode, err := database.GetAllWebhooks()
		if err != nil {
			err := "Failed to get all webhooks from database, got statuscode " + string(statusCode) + ", with error: " + err.Error()
			msg.Message = err
			println(err)
			ok := exchange.SendMessageWebhook(msg)
			if !ok {
				println("Error when sending message-webhook")
			}
			return false, currency
		}

		for i := 0; i < len(webhooks); i++ {
			rateCurrency := webhooks[i].TargetCurrency
			if currency.Base == webhooks[i].BaseCurrency && (currency.Rates[rateCurrency] >= webhooks[i].MinTriggerValue && currency.Rates[rateCurrency] <= webhooks[i].MaxTriggerValue) { // If currency are over or below this webhook-threshold
				statusCode, err := exchange.SendWebhookFunc(webhooks[i], currency.Rates[rateCurrency])
				if err != nil {
					err := "Failed to send webhook number " + string(i) + " from database (will not send any more webhooks if any), got statuscode " + string(statusCode) + ", with error: " + err.Error()
					msg.Message = err
					println(err)
					sendMessageCheckOK(msg)
					return false, currency
				}
			}
		}

		message := "Registered new currency to database with the date " + currency.Date
		msg.Heading = "Registered new currency!"
		msg.Message = message
		println(message)
		sendMessageCheckOK(msg)
		return true, currency
	}

	message := "Checked for new currencies, but there was no new currencies to register to database, tried to register currency with date " + currency.Date + ", which is the same as latest currency (which also has the date " + latestDate
	msg.Heading = "No new currencies!"
	msg.Message = message
	println(message)
	sendMessageCheckOK(msg)
	return true, currency // This is also something that might be expected
}

func databaseCred(test bool) *exchange.MongoDB {
	if test {
		return &exchange.MongoDB{
			DatabaseURL:            "mongodb://localhost",
			DatabaseName:           "herus-cloud-tecnologies",
			WebhookCollectionName:  "webhooks_v2_test",
			CurrencyCollectionName: "currencies_v2_test",
		}
	}
	return &exchange.MongoDB{
		DatabaseURL:            "mongodb://CloudFullAccess:full1916@ds227045.mlab.com:27045/herus-cloud-tecnologies",
		DatabaseName:           "herus-cloud-tecnologies",
		WebhookCollectionName:  "webhooks_v2",
		CurrencyCollectionName: "currencies_v2",
	}
}

func getAllCurrenciesFromExternalDatabase(database exchange.Storage, date string) bool {
	ok, currency := getCurrencyFromExternalDatabase(database, date, "EUR")
	if !ok {
		return false
	}

	println("Number of currencies in EUR is: " + strconv.Itoa(len(currency.Rates)))
	for k := range currency.Rates {
		println("Shall get currency " + k)
		ok, _ := getCurrencyFromExternalDatabase(database, date, k)
		if !ok {
			return false
		}
	}
	return true
}

func main() {
	exchange.DB = databaseCred(false)
	exchange.DB.Init()
	for {
		getAllCurrenciesFromExternalDatabase(exchange.DB, "latest")
		time.Sleep(time.Hour * 24)
	}
}

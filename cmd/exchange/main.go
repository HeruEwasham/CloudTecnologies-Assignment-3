package main

import "net/http"
import "github.com/HeruEwasham/CloudTecnologies-Assignment-3/exchange"
import "fmt"
import "os"

func databaseCred() *exchange.MongoDB {
	return &exchange.MongoDB{
		DatabaseURL:            "mongodb://CloudFullAccess:full1916@ds227045.mlab.com:27045/herus-cloud-tecnologies",
		DatabaseName:           "herus-cloud-tecnologies",
		WebhookCollectionName:  "webhooks_v2",
		CurrencyCollectionName: "currencies_v2",
	}
}

func main() {
	exchange.DB = databaseCred()

	exchange.DB.Init()
	//db.Init()
	http.HandleFunc("/exchange/latest", exchange.GetLatest)
	http.HandleFunc("/exchange/bot/latest", exchange.BotGetLatest)
	http.HandleFunc("/exchange/average", exchange.GetAverage)
	http.HandleFunc("/exchange/evaluationtrigger", exchange.EvaluationTrigger)
	http.HandleFunc("/exchange/", exchange.RegisterWebhook)
	http.HandleFunc("/exchange", exchange.RegisterWebhook) // Handle registration of a webhook.
	fmt.Println("Listen on Port:" + os.Getenv("PORT"))
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		panic(err)
	}
}

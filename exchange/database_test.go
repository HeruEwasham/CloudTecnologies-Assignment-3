package exchange

import "testing"
import "fmt"
import "time"
import "net/http"
import "net/http/httptest"
import "bytes"
import "encoding/json"

var testdb Storage

func setupTestdatabase() {
	testdb = &MongoDB{
		"mongodb://CloudFullAccess:full1916@ds227045.mlab.com:27045/herus-cloud-tecnologies",
		"herus-cloud-tecnologies",
		"webhooks_v2_test",
		"currencies_v2_test",
	}
}

func webhookIsLike(webhook1 Webhook, webhook2 Webhook) bool {
	// If anything is different between those two:
	if webhook1.BaseCurrency != webhook2.BaseCurrency || webhook1.MaxTriggerValue != webhook2.MaxTriggerValue || webhook1.MinTriggerValue != webhook2.MinTriggerValue || webhook1.TargetCurrency != webhook2.TargetCurrency || webhook1.WebhookURL != webhook2.WebhookURL {
		return false
	}
	// Else:
	return true
}

func Test_FloatToString(t *testing.T) {
	var input float32
	input = 2.35
	correctOutput := "2.35"
	output := FloatToString(input)
	if output != correctOutput {
		t.Error("Error when converting float to string, output is " + output + ", while correct output should be " + correctOutput)
	}
}

func Test_sendMsgWebhook(t *testing.T) {
	msg := MessageWebhook{"Test", time.Now().Format("2006-01-02-15:04:05"), "This is a test, if this is registered it worked", "Cloud tecnologies: Assignment 2"}
	ok := SendMessageWebhook(msg)
	if !ok {
		t.Error("Send Message didn't work")
	}
}

func Test_RegisterAndGetWebhook(t *testing.T) {
	setupTestdatabase() //?
	testdb.Init()       //?
	webhook := Webhook{WebhookURL: "http://example.com", BaseCurrency: "EUR", TargetCurrency: "NOK", MinTriggerValue: 1.5, MaxTriggerValue: 2.8}
	id, statusCode, err := testdb.RegisterWebhookToDatabase(webhook)
	if err != nil {
		t.Error("Error when registering webhook, statuscode is ", statusCode, ", and error is ", err)
		return
	}
	//if (id == "") {
	//	t.Error("ID is empty")
	//	return
	//}

	webhookGotten, statusCode, err := testdb.GetWebhook(id)
	if err != nil {
		t.Error("Error when getting webhook, id is: "+id+"statuscode is ", statusCode, ", and error is ", err)
		return
	}
	if !webhookIsLike(webhookGotten, webhook) {
		t.Error("base-webhook is different than the webhook gotten.")
		return
	}

	statusCode, err = SendWebhookFunc(webhookGotten, 1.8)
	if err != nil {
		t.Error("Something is wrong when sending webhook.")
	}

	webhooksGotten, statusCodeAll, errAll := testdb.GetAllWebhooks()
	if err != nil {
		t.Error("Error when getting all webhooks, statuscode is ", statusCodeAll, ", and error is ", errAll)
		return
	}
	fmt.Println(webhooksGotten)
	if !webhookIsLike(webhooksGotten[0], webhook) {
		t.Error("base-webhook is different than the webhook gotten when calling for all.")
		return
	}

	statusCode, err = testdb.DeleteWebhook(id)
	if err != nil {
		t.Error("Error when deleting webhook, statuscode is ", statusCode, ", and error is ", err)
		return
	}
	ok := testdb.ResetWebhook()
	if !ok {
		println("Couldn't reset Webhook-collection (connection-fault), manually deletion neccessarry")		// Will not give error, but just a remark since this is just to tidy up after the tests.
		return
	}
}

func Test_GetLatest(t *testing.T) {
	setupTestdatabase()                 //?
	testdb.Init()                       //?
	rateMap := make(map[string]float32) // Make a map with rates
	rateMap["NOK"] = 1.56
	currency := Currency{"EUR", "2100-01-01", rateMap}
	statusCode, err := testdb.RegisterCurrencyToDatabase(currency)
	if err != nil {
		t.Error("Error when registering currency, statuscode is ", statusCode, ", and error is ", err)
		return
	}
	latestCurrency, _, statusCode, err := testdb.GetLatest("EUR", "NOK") // Testing by getting latest Currency for Norwegian Kroner
	if err != nil {
		t.Error("Error when getting latest currency, statuscode is ", statusCode, ", and error is ", err)
		return
	}

	if latestCurrency != rateMap["NOK"] {
		t.Error("Latest currency gotten is not the one inserted just before, latest currency inserted is ", rateMap["NOK"], ", while we got ", latestCurrency)
		return
	}
	ok := testdb.ResetCurrency()
	if !ok {
		println("Couldn't reset Currency-collection (connection-fault), manually deletion neccessary")		// Will not give error, but just a remark since this is just to tidy up after the tests.
		return
	}
}

func Test_GetAverage(t *testing.T) {
	setupTestdatabase()                 //?
	testdb.Init()                       //?
	rateMap := make(map[string]float32) // Make a map with rates
	var rateAverage float32
	for i := 1; i <= 3; i++ {
		rateMap["NOK"] = 1.56 + float32(i)
		rateAverage += rateMap["NOK"]
		currency := Currency{"EUR", "2100-01-0" + string(i), rateMap}
		statusCode, err := testdb.RegisterCurrencyToDatabase(currency)
		if err != nil {
			t.Error("Error when registering currency number ", i, ", statuscode is ", statusCode, ", and error is ", err)
			return
		}
	}
	averageCurrrency, statusCode, err := testdb.GetAverage("EUR", "NOK") // Testing by getting latest Currency for Norwegian Kroner
	if err != nil {
		t.Error("Error when getting average currency, statuscode is ", statusCode, ", and error is ", err)
		return
	}

	if averageCurrrency != (rateAverage / 3) {
		t.Error("Average currency gotten is not the one inserted just before, latest currency inserted is ", rateAverage, ", while we got ", averageCurrrency)
		return
	}

	ok := testdb.ResetCurrency()

	if !ok {
		println("Error when resetting currency (connection-fault), manually deletion necessarry")		// Will not give error, but just a remark since this is just to tidy up after the tests.
	}
}

func Test_Handler_GetLatest(t *testing.T) {
	fmt.Println("Starting handler func")
	setupTestdatabase()                 //?
	testdb.Init()                       //?
	rateMap := make(map[string]float32) // Make a map with rates
	rateMap["NOK"] = 1.56
	currency := Currency{"EUR", "2100-01-01", rateMap}
	statusCode, err := testdb.RegisterCurrencyToDatabase(currency)
	if err != nil {
		t.Error("Error when registering currency, statuscode is ", statusCode, ", and error is ", err)
		return
	}
	ts := httptest.NewServer(http.HandlerFunc(BotGetLatest))
	defer ts.Close()
	postRequest := CurrencyRequest{}
	postRequest.BaseCurrency = "EUR"
	postRequest.TargetCurrency = "NOK"
	jsonStr := new(bytes.Buffer)
	json.NewEncoder(jsonStr).Encode(&postRequest)
	r := bytes.NewReader(jsonStr.Bytes())
	req, err := http.NewRequest("GET", ts.URL + "/exchange/bot/latest", r)
	if err != nil {
		t.Error("Error when trying to make a new request to botGetLatest-handler. Error: ", err)
		return
	}
	//defer req.Body.Close()
	// We can't have this set. And it only contains "/pkg/net/http/" anyway
    req.RequestURI = ""
	
	req.Close = true	
	//req, err := http.NewRequest("POST", url, )
	println("Handler GetLatest, json: " + jsonStr.String())
	//req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error("Error when trying to get response from getLatest-handler. Error: ", err)
		return
	}
	defer resp.Body.Close()
	println(resp.StatusCode)
	var responseBody[] byte
	req.Body.Read(responseBody)
	fmt.Println("Body gotten back from GetLatest-handler is " + string(responseBody))
	testdb.ResetCurrency()
}

/*func Test_Handler_GetLatestBot(t *testing.T) {
	fmt.Println("Starting handler func")
	setupTestdatabase()                 //?
	testdb.Init()                       //?
	rateMap := make(map[string]float32) // Make a map with rates
	rateMap["NOK"] = 1.56
	currency := Currency{"EUR", "2100-01-01", rateMap}
	statusCode, err := testdb.RegisterCurrencyToDatabase(currency)
	if err != nil {
		t.Error("Error when registering currency, statuscode is ", statusCode, ", and error is ", err)
		return
	}
	ts := httptest.NewServer(http.HandlerFunc(BotGetLatest))
	defer ts.Close()
	postRequest := CurrencyRequest{}
	postRequest.BaseCurrency = "EUR"
	postRequest.TargetCurrency = "NOK"
	jsonStr := new(bytes.Buffer)
	json.NewEncoder(jsonStr).Encode(&postRequest)
	r := bytes.NewReader(jsonStr.Bytes())
	req, err := http.NewRequest("POST", ts.URL + "/exchange/latest", r)
	if err != nil {
		t.Error("Error when trying to make a new request to getLatest-handler. Error: ", err)
		return
	}
	req.Body.Close()	
	//req, err := http.NewRequest("POST", url, )
	println("Handler GetLatest, json: " + jsonStr.String())
	//req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")
	
	// We can't have this set. And it only contains "/pkg/net/http/" anyway
    req.RequestURI = ""
	
	req.Close = true

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error("Error when trying to get response from getLatest-handler. Error: ", err)
	//println("error")
		return
	}
	defer resp.Body.Close()
	println(resp.StatusCode)
	var responseBody[] byte
	req.Body.Read(responseBody)
	fmt.Println("Body gotten back from GetLatest-handler is " + string(responseBody))
	testdb.ResetCurrency()
}

func Test_Handler_Webhook(t *testing.T) {
	fmt.Println("Starting handler func")
	setupTestdatabase()                 //?
	testdb.Init()                       //?
	ts := httptest.NewServer(http.HandlerFunc(RegisterWebhook))
	defer ts.Close()
	//jsonStr := new(bytes.Buffer)
	//json.NewEncoder(jsonStr).Encode(&postRequest)
	//r := bytes.NewReader(jsonStr.Bytes())
	req, err := http.NewRequest("GET", ts.URL + "/exchange/5a10291a14291b49ec19975d", nil)
	if err != nil {
		t.Error("Error when trying to make a new request to webhook-handler. Error: ", err)
		return
	}
	//req.Body.Close()	
	//req, err := http.NewRequest("POST", url, )
	//println("Handler registerWebhook, json: " + jsonStr.String())
	//req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")
	
	// We can't have this set. And it only contains "/pkg/net/http/" anyway
    //req.RequestURI = ""
	
	req.Close = true

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error("Error when trying to get response from webhook-handler. Error: ", err)
		return
	}
	defer resp.Body.Close()
	println(resp.StatusCode)
	var responseBody[] byte
	req.Body.Read(responseBody)
	//fmt.Println("Body gotten back from GetLatest-handler is " + string(responseBody))
	
}*/
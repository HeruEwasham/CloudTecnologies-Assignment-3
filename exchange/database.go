package exchange

import "fmt"
import "net/http"
import "strings"
import "encoding/json"
import "gopkg.in/mgo.v2"
import "gopkg.in/mgo.v2/bson"
import "errors"
import "bytes"
import "strconv"

// Webhook - This is the struct which will hold information about a webhook.
type Webhook struct {
	WebhookURL      string  `json:"webhookurl"`
	BaseCurrency    string  `json:"baseCurrency"`
	TargetCurrency  string  `json:"targetCurrency"`
	MinTriggerValue float32 `json:"minTriggerValue"`
	MaxTriggerValue float32 `json:"maxTriggerValue"`
	ID              string  `json:"id"`
}

// CurrencyRequest - This is the struct which will hold information about currencies from user.
type CurrencyRequest struct {
	BaseCurrency   string `json:"baseCurrency"`
	TargetCurrency string `json:"targetCurrency"`
}

// BotRequest - This is the struct which will hold information gotten from a bot (just useful stuff).
type BotRequest struct {
	Language 	string `json:"lang"`
	Status		struct {
		ErrorType	string `json:"errorType"`
		Code		float32  `json:"code"`
	} `json:"status"`
	Result   	struct {
		Parameters	struct {
			BaseCurrency   string `json:"baseCurrency"`
			TargetCurrency string `json:"targetCurrency"`
		} `json:"parameters"`
	} `json:"result"`
}

// BotAnswer - This is the struct which we will use to send a meaningful answer to a bot.
type BotAnswer struct {
	Speech   	string `json:"speech"`
	DisplayText string `json:"displayText"`
	Data   		struct {
		Argument string
		Message string
	} `json:"data"`
	//ContextOut   map `json:"contextOut"`
	Source   string `json:"source"`
}


// Currency - This is the struct which holds the currencies from
type Currency struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float32 `json:"rates"`
}

// SendWebhook - This is the struct which we use to make json to send
type SendWebhook struct {
	BaseCurrency    string  `json:"baseCurrency"`
	TargetCurrency  string  `json:"targetCurrency"`
	CurrentRate     float32 `json:"currentRate"`
	MinTriggerValue float32 `json:"minTriggerValue"`
	MaxTriggerValue float32 `json:"maxTriggerValue"`
}

// MessageWebhook - This is a struct which is used to send messages via webhook
type MessageWebhook struct {
	Heading     string `json:"heading"`
	DateTime    string `json:"dateTime"`
	Message     string `json:"message"`
	FromService string `json:"fromService"`
}

// Storage represent a unified way of getting webhook storage
type Storage interface {
	Init()
	RegisterWebhookToDatabase(webhook Webhook) (string, int, error)
	GetWebhook(id string) (Webhook, int, error)
	DeleteWebhook(id string) (int, error)
	GetLatest(string, string) (float32, string, int, error)
	GetAverage(string, string) (float32, int, error)
	RegisterCurrencyToDatabase(Currency) (int, error)
	GetAllWebhooks() ([]Webhook, int, error)
	ResetWebhook() bool
	ResetCurrency() bool
}

// MongoDB stores the database connection.
type MongoDB struct {
	DatabaseURL            string
	DatabaseName           string
	WebhookCollectionName  string
	CurrencyCollectionName string
}

// DB stores the database details
var DB Storage

// FloatToString - Convert float to string
func FloatToString(inputNum float32) string { // Gotten from: https://stackoverflow.com/questions/19101419/go-golang-formatfloat-convert-float-number-to-string
	// to convert a float number to a string
	return strconv.FormatFloat(float64(inputNum), 'f', -1, 32)
	//return fmt.Sprintf("%.6f", inputNum)
}

// Init initialize the MongoDB.
func (DB *MongoDB) Init() {
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()
	index := mgo.Index{
		Key:        []string{"date"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err = session.DB(DB.DatabaseName).C(DB.CurrencyCollectionName).EnsureIndex(index)
	if err != nil {
		panic(err)
	}
}

// RegisterWebhookToDatabase register a webhook to database.
func (DB *MongoDB) RegisterWebhookToDatabase(webhook Webhook) (string, int, error) {
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		return "", 500, err
	}
	defer session.Close()
	//wh := Webhook{}
	webhook.ID = bson.NewObjectId().Hex()
	err = session.DB(DB.DatabaseName).C(DB.WebhookCollectionName).Insert(&webhook)
	if err != nil {
		return "", 500, err
	}

	fmt.Println(webhook.ID)
	return webhook.ID, 201, nil
}

// RegisterCurrencyToDatabase register a webhook to database.
func (DB *MongoDB) RegisterCurrencyToDatabase(currency Currency) (int, error) {
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		return 500, err
	}
	defer session.Close()

	err = session.DB(DB.DatabaseName).C(DB.CurrencyCollectionName).Insert(&currency)
	if err != nil {
		return 500, err
	}

	return 201, nil
}

// GetWebhook gets the information of the webhook with id id.
func (DB *MongoDB) GetWebhook(id string) (Webhook, int, error) {
	webhook := Webhook{}
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		return webhook, 500, err
	}
	defer session.Close()

	err = session.DB(DB.DatabaseName).C(DB.WebhookCollectionName).Find(bson.M{"id": id}).One(&webhook)
	if err != nil {
		return webhook, http.StatusBadRequest, err // Suppose the reason for error here is because it can't find webhook with the user-sent id.
	}
	return webhook, http.StatusFound, nil
}

// DeleteWebhook deletes the webhook with id id.
func (DB *MongoDB) DeleteWebhook(id string) (int, error) {
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		return 500, err
	}
	defer session.Close()

	err = session.DB(DB.DatabaseName).C(DB.WebhookCollectionName).Remove(bson.M{"id": id})
	if err != nil {
		return http.StatusBadRequest, err // Suppose the reason for error here is because it can't find webhook with the user-sent id.
	}
	return http.StatusFound, nil
}

// GetLatest return the latest between currencies.
func (DB *MongoDB) GetLatest(baseCurrency string, targetCurrency string) (float32, string, int, error) {
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		return -1, "", 500, err
	}
	defer session.Close()
	latestCurrency := Currency{}
	dbSize, err := session.DB(DB.DatabaseName).C(DB.CurrencyCollectionName).Count()
	if err != nil {
		return -1, "", 500, err
	}
	err = session.DB(DB.DatabaseName).C(DB.CurrencyCollectionName).Find(bson.M{/*"baseCurrency": baseCurrency*/}).Sort("date").Skip(dbSize - 1).One(&latestCurrency) // Gotten from: https://stackoverflow.com/questions/38127583/get-last-inserted-element-from-mongodb-in-golang
	if err != nil {
		return -1, "", http.StatusBadRequest, err
	}
	val, ok := latestCurrency.Rates[targetCurrency] // This line + if sentence I have gotten from: https://stackoverflow.com/questions/2050391/how-to-check-if-a-map-contains-a-key-in-go
	if !ok {                                        // If an error, ie. targetCurrency do not exist.
		return -1, "", http.StatusBadRequest, errors.New("TargetCurrency not an accepted rate")
	}
	return val, latestCurrency.Date, http.StatusFound, nil
}

// GetAverage return the average between currencies the last 7 days.
func (DB *MongoDB) GetAverage(baseCurrency string, targetCurrency string) (float32, int, error) {
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		return -1, 500, err
	}
	defer session.Close()
	latestCurrencies := []Currency{}
	dbSize, err := session.DB(DB.DatabaseName).C(DB.CurrencyCollectionName).Count()
	if err != nil {
		return -1, 500, err
	}
	err = session.DB(DB.DatabaseName).C(DB.CurrencyCollectionName).Find(bson.M{"baseCurrency": baseCurrency}).Sort("date").Skip(dbSize - 3).All(&latestCurrencies) // Get the last 7 entries (last seven days). Gotten from: https://stackoverflow.com/questions/38127583/get-last-inserted-element-from-mongodb-in-golang and https://stackoverflow.com/questions/27165692/how-to-get-all-element-from-mongodb-array-using-go
	if err != nil {
		return -1, http.StatusBadRequest, err
	}
	var total float32
	for i := 0; i < 3; i++ {
		val, ok := latestCurrencies[i].Rates[targetCurrency] // This line + if sentence I have gotten from: https://stackoverflow.com/questions/2050391/how-to-check-if-a-map-contains-a-key-in-go
		if !ok {                                             // If an error, ie. targetCurrency do not exist.
			return -1, http.StatusBadRequest, errors.New("TargetCurrency not an accepted rate in " + string(i))
		}
		total += val
	}
	return total / 3, http.StatusFound, nil // Return average of the last seven days
}

// GetAllWebhooks gets all webhooks
func (DB *MongoDB) GetAllWebhooks() ([]Webhook, int, error) {
	webhooks := []Webhook{}
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		return webhooks, 500, err
	}
	defer session.Close()
	err = session.DB(DB.DatabaseName).C(DB.WebhookCollectionName).Find(bson.M{}).All(&webhooks) // Get all webhooks
	if err != nil {
		return webhooks, http.StatusBadRequest, err
	}
	return webhooks, 200, nil
}

// ResetWebhook deletes webhook-collection
func (DB *MongoDB) ResetWebhook() bool {
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		return false
	}
	err = session.DB(DB.DatabaseName).C(DB.WebhookCollectionName).DropCollection()
	if err != nil {
		return false
	}
	return true
}

// ResetCurrency deletes currency-collection
func (DB *MongoDB) ResetCurrency() bool {
	session, err := mgo.Dial(DB.DatabaseURL)
	if err != nil {
		return false
	}
	err = session.DB(DB.DatabaseName).C(DB.CurrencyCollectionName).DropCollection()
	if err != nil {
		return false
	}
	return true
}

// SendWebhookFunc sends info to a webhook.
func SendWebhookFunc(webhook Webhook, currentRate float32) (int, error) {
	//if currentRate >= webhook.MinTriggerValue && currentRate <= webhook.MaxTriggerValue {		// If you should send a webhook request (check a second/last time).
	sendWebhook := SendWebhook{}
	sendWebhook.BaseCurrency = webhook.BaseCurrency
	sendWebhook.CurrentRate = currentRate
	sendWebhook.MaxTriggerValue = webhook.MaxTriggerValue
	sendWebhook.MinTriggerValue = webhook.MinTriggerValue
	sendWebhook.TargetCurrency = webhook.TargetCurrency

	// This code gotten from: https://stackoverflow.com/questions/24455147/how-do-i-send-a-json-string-in-a-post-request-in-go
	jsonStr := new(bytes.Buffer)
	json.NewEncoder(jsonStr).Encode(&sendWebhook)
	//println(jsonString)
	//resp, err := http.Post(webhook.WebhookURL, "application/json", jsonStr)
	req, err := http.NewRequest("POST", webhook.WebhookURL, jsonStr)
	//req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return http.StatusExpectationFailed, err // http.StatusExpectationFailed hears out to be the best when not connecting to url (or something there)
	}
	println(resp.StatusCode)
	defer resp.Body.Close()
	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return http.StatusOK, nil // Has done the job without problems.
	}
	return http.StatusBadRequest, errors.New("We didn't get the correct statuscode, we got " + strconv.Itoa(resp.StatusCode) + ", but expected 200 or 204 when sending json ")
	//}
	//return http.StatusExpectationFailed, errors.New("Error when checking between values") // errors.New("Current rate is " + string(currentRate) + " which is between min and max, which is " + string(webhook.MinTriggerValue) + " and " + string(webhook.MaxTriggerValue))				// An error message if needed for debugging.
}

// SendMessageWebhook sends a message-webhook
func SendMessageWebhook(msg MessageWebhook) bool {
	// This code gotten from: https://stackoverflow.com/questions/24455147/how-do-i-send-a-json-string-in-a-post-request-in-go
	jsonStr := new(bytes.Buffer)
	json.NewEncoder(jsonStr).Encode(&msg)
	resp, err := http.Post("https://hooks.zapier.com/hooks/catch/2217946/if5wx8/", "application/json", jsonStr)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	/*if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return true					// Has done the job without problems.
	}
	return false*/
	return true
}

// RegisterWebhook registers, gets or deletes a webhook
func RegisterWebhook(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/") // Parts of response
	fmt.Println(parts)
	if len(parts) == 2 { // Is /exchange
		if r.Method == "POST" {
			webhook := Webhook{}
			decodeErr := json.NewDecoder(r.Body).Decode(&webhook) // Get POST-request
			if decodeErr != nil {
				http.Error(w, "Bad request: Decoding didn't work on POST-input. "+decodeErr.Error(), http.StatusBadRequest)
				return
			}
			if webhook.BaseCurrency != "EUR" { // If not implemented currency
				http.Error(w, "Not implemented: We accept just Euro as base-currency, while you gave us "+webhook.BaseCurrency, http.StatusNotImplemented)
				return
			}
			id, statusCode, err := DB.RegisterWebhookToDatabase(webhook)
			if err != nil {
				http.Error(w, "Something went wrong when registering a webhook: "+err.Error(), statusCode)
				return
			}
			http.Header.Add(w.Header(), "content-type", "text")
			w.WriteHeader(statusCode)
			fmt.Fprintf(w, string(id))
		} else {
			http.Error(w, "Method not allowed: We only support POST for this functionality, but you used "+r.Method+".", http.StatusMethodNotAllowed)
			return
		}
	} else if len(parts) == 3 { // Is /exchange/{id}
		if r.Method == "GET" {
			webhook, statusCode, err := DB.GetWebhook(parts[2]) // Get webhook by id from user
			if err != nil {
				http.Error(w, "Something went wrong when getting a webhook: "+err.Error(), statusCode)
				return
			}
			http.Header.Add(w.Header(), "content-type", "application/json")
			json.NewEncoder(w).Encode(&webhook)

		} else if r.Method == "DELETE" {
			statusCode, err := DB.DeleteWebhook(parts[2]) // Delete webhook with id gotten by user
			if err != nil {
				http.Error(w, "Something went wrong when deleting a webhook: "+err.Error(), statusCode)
				return
			}

		} else {
			http.Error(w, "Method not allowed: We only support GET and DELETE for this functionality, but you used "+r.Method+".", http.StatusMethodNotAllowed)
			return
		}
	} else {
		http.Error(w, "Bad request: You didn't give us enough/correct arguments", http.StatusBadRequest)
		return
	}
}

// GetLatest gets the lates currency (in date-order)
func GetLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		currencyRequest := CurrencyRequest{}
		decodeErr := json.NewDecoder(r.Body).Decode(&currencyRequest) // Get POST-request
		if decodeErr != nil {
			http.Error(w, "Bad request: Decoding didn't work on POST-input. "+decodeErr.Error(), http.StatusBadRequest)
			return
		}
		/*if currencyRequest.BaseCurrency != "EUR" {
			http.Error(w, "Not implemented: We only support Euro as baseCurrency. ", http.StatusNotImplemented)
			return
		}*/

		latestCurrency, _, statusCode, err := DB.GetLatest(currencyRequest.BaseCurrency, currencyRequest.TargetCurrency) // Get latest currency from database
		if err != nil {
			http.Error(w, "We got an error while getting latest currency: "+err.Error(), statusCode)
			return
		}

		http.Header.Add(w.Header(), "content-type", "text")
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, FloatToString(latestCurrency))

	} else {
		http.Error(w, "Method not allowed: We only support POST for this functionality, but you used "+r.Method+".", http.StatusNotImplemented)
		return
	}
}

// GetAverage gets the average of the last 3 currencies
func GetAverage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		currencyRequest := CurrencyRequest{}
		decodeErr := json.NewDecoder(r.Body).Decode(&currencyRequest) // Get POST-request
		if decodeErr != nil {
			http.Error(w, "Bad request: Decoding didn't work on POST-input. "+decodeErr.Error(), http.StatusBadRequest)
			return
		}
		/*if currencyRequest.BaseCurrency != "EUR" {
			http.Error(w, "Not implemented: We only support Euro as baseCurrency. ", http.StatusNotImplemented)
			return
		}*/

		averageCurrency, statusCode, err := DB.GetAverage(currencyRequest.BaseCurrency, currencyRequest.TargetCurrency) // Get the average currency from database
		if err != nil {
			http.Error(w, "We got an error while getting average currency: "+err.Error(), statusCode)
			return
		}

		// TODO: Give output to user
		http.Header.Add(w.Header(), "content-type", "text")
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, FloatToString(averageCurrency))

	} else {
		http.Error(w, "Not implemented: We only support POST for this functionality.", http.StatusNotImplemented)
		return
	}
}

// EvaluationTrigger triggers all webhooks to be sent
func EvaluationTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		webhooks, statusCode, err := DB.GetAllWebhooks()
		if err != nil {
			http.Error(w, "Failed to get all webhooks from database. Error: "+err.Error(), statusCode)
			return
		}

		for i := 0; i < len(webhooks); i++ {
			//fmt.Fprint(w, string(webhooks[i].WebhookURL))
			baseCurrency := webhooks[i].BaseCurrency
			rateCurrency := webhooks[i].TargetCurrency
			latestCurrency, _, statusCode, err := DB.GetLatest(baseCurrency, rateCurrency) // Get latest currency from database
			statusCode, err = SendWebhookFunc(webhooks[i], latestCurrency)   // Sends latest currency
			if err != nil {
				http.Error(w, "Failed to send webhook number "+strconv.Itoa(i)+" from database (will not send any more webhooks if any). Error:"+err.Error(), statusCode)
				return
			}
		}

		w.WriteHeader(http.StatusOK)

	} else {
		http.Error(w, "Not implemented: We only support GET here", http.StatusMethodNotAllowed)
	}
}

// BotGetLatest gets the lates currency (in date-order) and sends a bot-understandable json as answer.
func BotGetLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		botRequest := BotRequest{}
		decodeErr := json.NewDecoder(r.Body).Decode(&botRequest) // Get POST-request
		if decodeErr != nil {
			http.Error(w, "Bad request: Decoding didn't work on POST-input. "+decodeErr.Error(), http.StatusBadRequest)
			return
		}
		if botRequest.Result.Parameters.BaseCurrency != "EUR" {
			http.Error(w, "Not implemented: We only support Euro as baseCurrency. ", http.StatusNotImplemented)
			return
		}

		latestCurrency, _, statusCode, err := DB.GetLatest(botRequest.Result.Parameters.BaseCurrency, botRequest.Result.Parameters.TargetCurrency) // Get latest currency from database
		if err != nil {
			http.Error(w, "We got an error while getting latest currency: "+err.Error(), statusCode)
			return
		}

		response := "The latest currency conversion between " + botRequest.Result.Parameters.BaseCurrency + " and " + botRequest.Result.Parameters.TargetCurrency + " is " + FloatToString(latestCurrency)
		botAnswer := BotAnswer{}
		botAnswer.Speech = response
		botAnswer.DisplayText = response

		// Return answer to bot:
		http.Header.Add(w.Header(), "content-type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(&botAnswer)

	} else {
		http.Error(w, "Method not allowed: We only support POST for this functionality, but you used "+r.Method+".", http.StatusNotImplemented)
		return
	}
}
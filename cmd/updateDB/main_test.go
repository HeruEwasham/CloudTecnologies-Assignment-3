package main

import "github.com/HeruEwasham/CloudTecnologies-Assignment-3/exchange"
import "testing"

var testdb exchange.Storage
var normaldb exchange.Storage

func setupNormalDatabase() {
	normaldb = databaseCred(false)
}
func setupTestdatabase() {
	testdb = databaseCred(true)
}

func Test_GetTodaysCurrency(t *testing.T) {
	setupNormalDatabase() // Test this part
	setupTestdatabase()   //?
	testdb.Init()         //?
	ok := getCurrencyFromExternalDatabase(testdb, "latest")
	if !ok {
		t.Error("Function getTodaysCurrency(..) failed. Most likely connection fault, try again.")
		return
	}

	ok = testdb.ResetCurrency()
	if !ok {
		println("Couldn't reset Currency-collection (connection fault), manually reset neccessarry")	// Will not give error, but just a remark since this is just to tidy up after the tests.
		return
	}
}

package rest

import (
	"testing"
)

func TestDatabaseConnector(test *testing.T) {
	test.Log("Checking Database connector...");
	db, err := CreateConnector();
	if err != nil {
		test.Error(err);

	}
	test.Log("Checking connection sanity...");
	result, err := db.Query("SELECT @@version"); //check that Database has return value
	if err != nil {
		test.Error(err);
	} else {
		test.Log(result);
	}

}

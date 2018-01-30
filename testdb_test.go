// Package testdb_test provides tests for testdb.
package testdb_test

import (
	"os"
	"testing"
	"time"

	"github.com/mongo-go/testdb"
	"gopkg.in/mgo.v2"
)

var defaultUrl = "localhost"
var defaultDb = "test"
var defaultTimeout = time.Duration(2) * time.Second

// A quick smoke test to make sure the basics work.
func TestTestDB(t *testing.T) {
	testDb := testdb.NewTestDB(defaultUrl, defaultDb, defaultTimeout)

	// CreateRandomCollection errors if called before Connect.
	_, err := testDb.CreateRandomCollection(&mgo.CollectionInfo{}, []mgo.Index{})
	if err == nil {
		t.Error("expected an error, did not get one")
	}

	// Connect to the db.
	if err = testDb.Connect(); err != nil {
		t.Fatal(err)
	}
	defer testDb.Close()

	// Calling OverrideWithEnvVars after Connect does nothing.
	testDb.OverrideWithEnvVars()

	indexes := []mgo.Index{
		{
			Key:    []string{"iamunique"},
			Unique: true,
		},
	}

	// Create a collection with a unique index.
	c, err := testDb.CreateRandomCollection(&mgo.CollectionInfo{}, indexes)
	if err != nil {
		t.Error(err)
	}

	defer func() {
		err := testDb.DropCollection(c)
		if err != nil {
			t.Error(err)
		}
	}()

	doc := map[string]string{
		"iamunique": "a",
	}

	err = c.Insert(doc)
	if err != nil {
		t.Error(err)
	}

	// Should get a duplicate key error when we try to insert the same doc.
	err = c.Insert(doc)
	if !mgo.IsDup(err) {
		t.Errorf("expected a duplicate key error, did not get one (err: %s)", err)
	}
}

func TestCreateCollectionInvalidIndex(t *testing.T) {
	testDb := testdb.NewTestDB(defaultUrl, defaultDb, defaultTimeout)

	if err := testDb.Connect(); err != nil {
		t.Fatal(err)
	}
	defer testDb.Close()

	// An invalid index.
	indexes := []mgo.Index{
		{
			Key:    []string{"", "", ""},
			Unique: true,
		},
	}

	_, err := testDb.CreateRandomCollection(&mgo.CollectionInfo{}, indexes)
	if err == nil {
		t.Error("expected an error, did not get one")
	}
}

func TestDropCollectionError(t *testing.T) {
	testDb := testdb.NewTestDB(defaultUrl, defaultDb, defaultTimeout)

	if err := testDb.Connect(); err != nil {
		t.Fatal(err)
	}
	defer testDb.Close()

	// Create a collection.
	c, err := testDb.CreateRandomCollection(&mgo.CollectionInfo{}, []mgo.Index{})
	if err != nil {
		t.Error(err)
	}

	// Drop the collection on our own without using TestDB.
	err = c.DropCollection()
	if err != nil {
		t.Error(err)
	}

	//	 Dropping the collection through TestDB should throw an error.
	err = testDb.DropCollection(c)
	if err == nil {
		t.Error("expected an error, did not get one")
	}
}

func TestEnvVarOverride(t *testing.T) {
	url := "jibberish:99999999999"
	db := "another_db"
	testDb := testdb.NewTestDB(url, db, defaultTimeout)

	if err := os.Setenv(testdb.ENV_VAR_TEST_MONGO_URL, defaultUrl); err != nil {
		t.Error(err)
	}
	if err := os.Setenv(testdb.ENV_VAR_TEST_MONGO_DB, defaultDb); err != nil {
		t.Error(err)
	}

	testDb.OverrideWithEnvVars()
	err := testDb.Connect()
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidUrl(t *testing.T) {
	url := "thisis?invalid"
	db := "test"
	testDb := testdb.NewTestDB(url, db, defaultTimeout)

	err := testDb.Connect()
	if err == nil {
		t.Fatal("expected an error, did not get one")
	}
}

func TestConnectError(t *testing.T) {
	url := "jibberish:99999999999"
	db := "test"
	timeout := time.Duration(100) * time.Millisecond

	testDb := testdb.NewTestDB(url, db, timeout)

	err := testDb.Connect()
	if err == nil {
		t.Fatal("expected an error, did not get one")
	}
}

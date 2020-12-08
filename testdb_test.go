// Package testdb_test provides tests for testdb.
package testdb_test

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mongo-go/testdb"
)

var (
	defaultUrl     = "mongodb://localhost"
	defaultDb      = "mongo-go"
	defaultTimeout = time.Duration(2) * time.Second
)

// A quick smoke test to make sure the basics work.
func TestTestDB(t *testing.T) {
	testDb := testdb.NewTestDB(defaultUrl, defaultDb, defaultTimeout)

	// CreateRandomCollection errors if called before Connect.
	coll, err := testDb.CreateRandomCollection(testdb.NoIndexes)
	if err == nil {
		t.Fatal("expected an error, did not get one")
	}

	// Connect to the db.
	if err = testDb.Connect(); err != nil {
		t.Fatal(err)
	}
	defer testDb.Close()

	// Calling OverrideWithEnvVars after Connect does nothing.
	testDb.OverrideWithEnvVars()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"iamunique", 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	// Create a collection with a unique index.
	coll, err = testDb.CreateRandomCollection(indexes)
	if err != nil {
		t.Error(err)
	}
	defer coll.Drop(context.Background())

	doc := map[string]string{
		"iamunique": "a",
	}

	_, err = coll.InsertOne(context.Background(), doc)
	if err != nil {
		t.Error(err)
	}

	// Should get a duplicate key error when we try to insert the same doc.
	_, err = coll.InsertOne(context.Background(), doc)
	if !testdb.IsDupeKeyError(err) {
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
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"", 1}},
		},
	}

	coll, err := testDb.CreateRandomCollection(indexes)
	if err == nil {
		t.Error("expected an error, did not get one")
	}
	if coll != nil {
		coll.Drop(context.Background())
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

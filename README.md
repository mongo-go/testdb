# Test databases for MongoDB, made easy.

[![Build Status](https://travis-ci.org/mongo-go/testdb.svg?branch=master)](https://travis-ci.org/mongo-go/testdb)
[![Go Report Card](https://goreportcard.com/badge/github.com/mongo-go/testdb)](https://goreportcard.com/report/github.com/mongo-go/testdb)
[![Coverage Status](https://coveralls.io/repos/github/mongo-go/testdb/badge.svg?branch=master&)](https://coveralls.io/github/mongo-go/testdb?branch=master)
[![GoDoc](https://godoc.org/github.com/mongo-go/testdb?status.svg)](https://pkg.go.dev/github.com/mongo-go/testdb)

This is a small Go package that makes it easy to create databases/collections for MongoDB tests.
It's useful when you want to run tests against an actual MongoDB instance.

## Setup
Install the package with "go get".
```
go get "github.com/mongo-go/testdb"
```

This package should only be imported in tests; you should never use it in actual code.

## Usage
Here is an example of how to use this package:
```go
package main_test

import (
        "testing"

        "github.com/mongo-go/testdb"
)

var testDb *testdb.TestDB

func setup(t *testing.T) *mgo.Collection {
	if testDb == nil {
		testDb = testdb.NewTestDB("mongodb://localhost", "your_db", time.Duration(2) * time.Second)

		err := testDb.Connect()
		if err != nil {
			t.Fatal(err)
		}
        }

        coll, err := testDb.CreateRandomCollection(testdb.NoIndexes)
        if err != nil {
                t.Fatal(err)
        }

        return coll // random *mongo.Collection in "your_db"
}

func Test1(t *testing.T) {
        coll := setup(t)
	defer coll.Drop(context.Background())

        // Test queries using coll
}
```

## Overriding Defaults with Environement Variables
One of the benefits of using this package is that it allows you to override certain defaults with environment variables.
These are the env vars currently supported:
* `TEST_MONGO_URL`: overrides the url of the MongoDB instance being used for testing.
* `TEST_MONGO_DB`: overrides the database name being used for testing.

By default, even if these env vars are set, they will not be used. To use them, you must call the OverrideWithEnvVars on a TestDB before calling Connect, like so:
```
// export TEST_MONGO_URL="their_url"
// export TEST_MONGO_DB="their_db"

testDb := testdb.NewTestDB("mongodb://localhost", "your_db", time.Duration(2) * time.Second)
testDb.OverrideWithEnvVars()

err := testdb.Conect() {
        // ...
}
```

Why is this useful? Say you have some tests that connect to MongoDB, which you always have running locally at "mongodb://localhost".
Hardcoding your tests to create a TestDB with that url works fine for you, but what about when someone else who has MongoDB running locally at "their_url" tries to run your tests?
By calling OverrideWithEnvVars in your tests you give whoever is invoking them the ability to change the url and database of the TestDB without having to change any code.

## Tests
Tests for this package require an instance of MongoDB to be running at "localhost" (no port).
They write into the db "test" and collection "testdb_collection", and delete all documents from that collection after they run.

Run the tests from the root directory of this repo like so:
```
go test `go list ./... | grep -v "/vendor/"` --race
```

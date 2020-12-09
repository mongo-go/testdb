// Package testdb provides the ability to easily create MongoDB databases/
// collections within tests.
package testdb

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// ENV_VAR_TEST_MONGO_URL is an environment variable that, if set, can
	// override the MongoDB url used in a TestDB. The OverrideWithEnvVars
	// method must be called for it to take effect.
	ENV_VAR_TEST_MONGO_URL = "TEST_MONGO_URL"

	// ENV_VAR_TEST_MONGO_DB is an environment variable that, if set, can
	// override the MongoDB database used in a TestDB. The OverrideWithEnvVars
	// method must be called for it to take effect.
	ENV_VAR_TEST_MONGO_DB = "TEST_MONGO_DB"
)

// NoIndexes can be passed to CreateRandomCollection to create a collection
// without indexes.
var NoIndexes []mongo.IndexModel

func init() { rand.Seed(time.Now().UnixNano()) }

// A TestDB represents a MongoDB database used for running tests against.
type TestDB struct {
	url     string
	db      string
	timeout time.Duration
	// --
	client *mongo.Client
}

// NewTestDB creates a new TestDB with the provided url, database name, and
// timeout. It doesn't actually connect to MongoDB; the Connect method must be
// called to do that.
func NewTestDB(url, db string, timeout time.Duration) *TestDB {
	return &TestDB{
		url:     url,
		db:      db,
		timeout: timeout,
	}
}

// OverrideWithEnvVars overrides the url and database in a TestDB if certain
// environment variables are set. This makes it easy for multiple people to
// run tests that require a MongoDB instance even if they have it running at
// different urls or if they want to use different databases.
//
// This method will only do anything if Connect hasn't already been called on
// the TestDB.
func (t *TestDB) OverrideWithEnvVars() {
	if t.client != nil {
		return
	}

	if urlOverride := os.Getenv(ENV_VAR_TEST_MONGO_URL); urlOverride != "" {
		t.url = urlOverride
	}
	if dbOverride := os.Getenv(ENV_VAR_TEST_MONGO_DB); dbOverride != "" {
		t.db = dbOverride
	}
}

// Connect initializes a connection to the TestDB. It will return an error if
// it cannot connect to MongoDB.
func (t *TestDB) Connect() error {
	// SetServerSelectionTimeout is different and more important than SetConnectTimeout.
	// Internally, the mongo driver is polling and updating the topology,
	// i.e. the list of replicas/nodes in the cluster. SetServerSelectionTimeout
	// applies to selecting a node from the topology, which should be nearly
	// instantaneous when the cluster is ok _and_ when it's down. When a node
	// is down, it's reflected in the topology, so there's no need to wait for
	// another server because we only use one server: the master replica.
	// The 500ms below is really how long the driver will wait for the master
	// replica to come back online.
	//
	// SetConnectTimeout is what is seems: timeout when a connection is actually
	// made. This guards against slows networks, or the case when the mongo driver
	// thinks the master is online but really it's not.
	opts := options.Client().
		ApplyURI(t.url).
		SetConnectTimeout(t.timeout).
		SetServerSelectionTimeout(time.Duration(500 * time.Millisecond))

	client, err := mongo.NewClient(opts)
	if err != nil {
		return err
	}

	// mongo.Connect() does not actually connect:
	//   The Client.Connect method starts background goroutines to monitor the
	//   state of the deployment and does not do any I/O in the main goroutine to
	//   prevent the main goroutine from blocking. Therefore, it will not error if
	//   the deployment is down.
	// https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo?tab=doc#Connect
	// The caller must call client.Ping() to actually connect. Consequently,
	// we don't need a context here. As long as there's not a bug in the mongo
	// driver, this won't block.
	if err := client.Connect(context.Background()); err != nil {
		return err
	}

	t.client = client
	return nil
}

// CreateRandomCollection creates a collection with the details of info, and
// ensures it has the provided indexes. The name of the collection will be
// random, following the format of "test_" + 8 random characters. The
// DropCollection method should always be called to clean up collections
// created by this method.
//
// TestDB only supports creating random collections due to the fact that tests
// run concurrently. If multiple tests used the same collection, they would
// probably stomp on each other.
func (t *TestDB) CreateRandomCollection(indexes []mongo.IndexModel) (*mongo.Collection, error) {
	if t.client == nil {
		return nil, fmt.Errorf("must call Connect first")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := "test_" + randSeq(8)
	coll := t.client.Database(t.db).Collection(collection)

	if len(indexes) > 0 {
		indexView := coll.Indexes()

		opts := options.CreateIndexes().SetMaxTime(2 * time.Second)
		if _, err := indexView.CreateMany(ctx, indexes, opts); err != nil {
			coll.Drop(ctx)
			return nil, err
		}
	}

	return coll, nil
}

// Close terminates the TestDB's connection to MongoDB.
func (t *TestDB) Close() {
	t.client.Disconnect(context.Background())
}

const dupeKeyCode = 11000

// IsDupeKeyError returns true if the error is a Mongo duplicate key error.
func IsDupeKeyError(err error) bool {
	// mongo.WriteException{
	//   WriteConcernError:(*mongo.WriteConcernError)(nil),
	//   WriteErrors:mongo.WriteErrors{
	//     mongo.WriteError{
	//       Index:0,
	//       Code:11000,
	//       Message:"E11000 duplicate key error collection: coll.nodes index: x_1 dup key: { : 6 }"
	//     }
	//   }
	// }
	if _, ok := err.(mongo.WriteException); ok {
		we := err.(mongo.WriteException)
		for _, e := range we.WriteErrors {
			if e.Code == dupeKeyCode {
				return true
			}
		}
	}
	if _, ok := err.(mongo.CommandError); ok {
		ce := err.(mongo.CommandError)
		if ce.Code == dupeKeyCode {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------------------- //

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

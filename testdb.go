// Package testdb provides the ability to easily create MongoDB databases/
// collections within tests.
package testdb

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"gopkg.in/mgo.v2"
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

func init() { rand.Seed(time.Now().UnixNano()) }

// A TestDB represents a MongoDB database used for running tests against.
type TestDB struct {
	url     string
	db      string
	timeout time.Duration
	// --
	session *mgo.Session
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
	if t.session != nil {
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
	dialInfo, err := mgo.ParseURL(t.url)
	if err != nil {
		return fmt.Errorf("error connecting to mongo on '%s': %s", t.url, err)
	}

	dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
		conn, err := net.DialTimeout("tcp", addr.String(), t.timeout)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
	dialInfo.Timeout = t.timeout

	s, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return fmt.Errorf("error connecting to mongo on '%s': %s", t.url, err)
	}

	t.session = s
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
func (t *TestDB) CreateRandomCollection(info *mgo.CollectionInfo, indexes []mgo.Index) (*mgo.Collection, error) {
	if t.session == nil {
		return nil, fmt.Errorf("must call Connect first")
	}

	s := t.session.Copy()

	collection := "test_" + randSeq(8)
	c := s.DB(t.db).C(collection)

	err := c.Create(info)
	if err != nil {
		return c, fmt.Errorf("error creating collection: %s", err)
	}

	for _, index := range indexes {
		err = c.EnsureIndex(index)
		if err != nil {
			return c, fmt.Errorf("error creating index '%#v': %s", index.Key, err)
		}
	}

	return c, nil
}

// DropCollection removes a collection and all of its documents from the
// TestDB. It also closes the MongoDB connection behind the collection. This
// method should always be called to clean up a collection created by the
// CreateRandomCollection method.
func (t *TestDB) DropCollection(c *mgo.Collection) error {
	err := c.DropCollection()
	if err != nil {
		return fmt.Errorf("error dropping the collection '%s': %s", c.FullName, err)
	}

	c.Database.Session.Close()
	return nil
}

// Close terminates the TestDB's connection to MongoDB.
func (t *TestDB) Close() {
	t.session.Close()
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

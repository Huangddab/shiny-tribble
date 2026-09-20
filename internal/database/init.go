package database

import (
	"context"
	"errors"
	"sync"

	"github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	_db  *Database
	once sync.Once
)

// InitializedDatabase initializes the database (singleton pattern)
func InitializedDatabase(ctx context.Context) {
	once.Do(func() {
		var err error
		if _db, err = NewDatabase(); err != nil {
			_db = nil
			logrus.Warnf("database is unavailable: %v", err)
		}
	})
}

// GetDatabase returns the singleton database instance
func GetDatabase() *Database {
	return _db
}

// Stop stops the database and closes the connection
func Stop() {
	if _db == nil {
		return
	}
	_db.Close()
}

// InsertOne inserts one document
func InsertOne(col string, document any) (*mongo.InsertOneResult, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.InsertOne(col, document)
}

// InsertMany inserts multiple documents
func InsertMany(col string, documents []any) (*mongo.InsertManyResult, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.InsertMany(col, documents)
}

// FindOne finds one document
func FindOne(col string, filter any) *mongo.SingleResult {
	if _db == nil {
		return nil
	}
	return _db.FindOne(col, filter)
}

// Find finds multiple documents
func Find(col string, filter any) (*mongo.Cursor, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.Find(col, filter)
}

func FindAll(col string, filter any, result any, opts ...*options.FindOptions) error {
	if _db == nil {
		return errors.New("database not initialized")
	}
	return _db.FindAll(col, filter, result, opts...)
}

// UpdateOne updates one document
func UpdateOne(col string, filter any, update any, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.UpdateOne(col, filter, update, opts...)
}

// UpdateMany updates multiple documents
func UpdateMany(col string, filter any, update any) (*mongo.UpdateResult, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.UpdateMany(col, filter, update)
}

// UpdateByID updates a document by ID
func UpdateByID(col string, id any, update any) (*mongo.UpdateResult, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.UpdateByID(col, id, update)
}

// DeleteOne deletes one document
func DeleteOne(col string, filter any) (*mongo.DeleteResult, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.DeleteOne(col, filter)
}

// DeleteMany deletes multiple documents
func DeleteMany(col string, filter any) (*mongo.DeleteResult, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.DeleteMany(col, filter)
}

// CountDocuments counts documents
func CountDocuments(col string, filter any) (int64, error) {
	if _db == nil {
		return 0, errors.New("database not initialized")
	}
	return _db.CountDocuments(col, filter)
}

// Aggregate performs  aggregation
func Aggregate(col string, pipeline any) (*mongo.Cursor, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.Aggregate(col, pipeline)
}

// CreateIndexes creates indexes
func CreateIndexes(col string, models []mongo.IndexModel) ([]string, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.CreateIndexes(col, models)
}

// BulkWrite performs bulk write
func BulkWrite(col string, models []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	if _db == nil {
		return nil, errors.New("database not initialized")
	}
	return _db.BulkWrite(col, models)
}

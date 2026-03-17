package database

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Database wraps MongoDB database connection and provides helper methods
type Database struct {
	*mongo.Database
}

// NewDatabase creates a new database connection
func NewDatabase() (*Database, error) {
	// Get configuration from viper with defaults
	dbURI := viper.GetString("database.uri")
	opts := options.Client().ApplyURI(dbURI)
	dbName := viper.GetString("database.dbname")

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logrus.Infof("Connecting to MongoDB: %s/%s", dbURI, dbName)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		logrus.Errorf("Failed to connect to %s: %v", dbURI, err)
		return nil, err
	}

	// Ping database to verify connection
	if err = client.Ping(ctx, nil); err != nil {
		logrus.Errorf("Failed to ping MongoDB: %v", err)
		client.Disconnect(ctx)
		return nil, err
	}

	return &Database{
		Database: client.Database(dbName),
	}, nil
}

// Close disconnects from MongoDB
func (db *Database) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.Client().Disconnect(ctx)
	if err != nil {
		logrus.Errorf("Failed to disconnect from MongoDB: %v", err)
		return err
	}

	logrus.Info("Disconnected from MongoDB")
	return nil
}

// InsertOne inserts a single document
func (db *Database) InsertOne(col string, document any, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).InsertOne(ctx, document, opts...)
}

// InsertMany inserts multiple documents
func (db *Database) InsertMany(col string, documents []any, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).InsertMany(ctx, documents, opts...)
}

// FindOne finds a single document
func (db *Database) FindOne(col string, filter any, opts ...*options.FindOneOptions) *mongo.SingleResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).FindOne(ctx, filter, opts...)
}

// Find finds multiple documents
func (db *Database) Find(col string, filter any, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).Find(ctx, filter, opts...)
}

// UpdateOne updates a single document
func (db *Database) UpdateOne(col string, filter any, update any) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).UpdateOne(ctx, filter, update)
}

// UpdateMany updates multiple documents
func (db *Database) UpdateMany(col string, filter any, update any) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).UpdateMany(ctx, filter, update)
}

// UpdateByID updates a document by ID
func (db *Database) UpdateByID(col string, id any, update any) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).UpdateByID(ctx, id, update)
}

// DeleteOne deletes a single document
func (db *Database) DeleteOne(col string, filter any) (*mongo.DeleteResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).DeleteOne(ctx, filter)
}

// DeleteMany deletes multiple documents
func (db *Database) DeleteMany(col string, filter any) (*mongo.DeleteResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).DeleteMany(ctx, filter)
}

// CountDocuments counts documents in a collection
func (db *Database) CountDocuments(col string, filter any, opts ...*options.CountOptions) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).CountDocuments(ctx, filter, opts...)
}

// Aggregate performs aggregation pipeline
func (db *Database) Aggregate(col string, pipeline any, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).Aggregate(ctx, pipeline, opts...)
}

// CreateIndexes creates indexes on a collection
func (db *Database) CreateIndexes(col string, models []mongo.IndexModel, opts ...*options.CreateIndexesOptions) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).Indexes().CreateMany(ctx, models, opts...)
}

// BulkWrite performs bulk write operations
func (db *Database) BulkWrite(col string, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Collection(col).BulkWrite(ctx, models, opts...)
}

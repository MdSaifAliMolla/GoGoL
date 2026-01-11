package storage

import (
	"context"
	"time"

	"github.com/MdSaifAliMolla/GoGoL/internal/crawler"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStorage struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewMongoStorage(uri, dbName, collName string) (*MongoStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	coll := client.Database(dbName).Collection(collName)
	return &MongoStorage{
		client:     client,
		collection: coll,
	}, nil
}

func (s *MongoStorage) SavePage(p crawler.Page) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.collection.InsertOne(ctx, p)
	return err
}

func (s *MongoStorage) Close() {
	if s.client != nil {
		s.client.Disconnect(context.Background())
	}
}

func (s *MongoStorage) GetPages() ([]crawler.Page, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := s.collection.Find(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var pages []crawler.Page
	if err := cursor.All(ctx, &pages); err != nil {
		return nil, err
	}
	return pages, nil
}


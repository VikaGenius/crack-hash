package repository

import (
	"context"
	"manager/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type MongoRepository struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

func NewMongoRepository(uri string) (*MongoRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wc := writeconcern.Majority()
    rc := readconcern.Majority()

    clientOpts := options.Client().
        ApplyURI(uri).
        SetWriteConcern(wc).
        SetReadConcern(rc)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	// Проверка подключения
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	// Создание коллекции с настройками
	db := client.Database("crackhash")
	collectionOpts := options.Collection().
		SetWriteConcern(wc).
		SetReadConcern(rc)

	collection := db.Collection("requests", collectionOpts)

	// Создание индексов
	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "request_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "updated_at", Value: 1}},
		},
	}

	_, err = collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return nil, err
	}

	return &MongoRepository{
		client:     client,
		db:         db,
		collection: collection,
	}, nil
}

func (r *MongoRepository) SaveRequest(ctx context.Context, req *models.CrackRequest) error {
	_, err := r.collection.InsertOne(ctx, req)
	return err
}

func (r *MongoRepository) GetRequest(ctx context.Context, requestID string) (*models.CrackRequest, error) {
	var req models.CrackRequest
	filter := bson.M{"request_id": requestID}
	err := r.collection.FindOne(ctx, filter).Decode(&req)
	return &req, err
}

func (r *MongoRepository) UpdateRequest(ctx context.Context, req *models.CrackRequest) error {
	filter := bson.M{"request_id": req.RequestID}
	update := bson.M{
		"$set": bson.M{
			"status":     req.Status,
			"data":       req.Data,
			"progress":   req.Progress,
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoRepository) GetPendingRequests(ctx context.Context) ([]models.CrackRequest, error) {
	filter := bson.M{
		"status": bson.M{
			"$in": []string{"IN_PROGRESS", "PARTIALLY_READY"},
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var requests []models.CrackRequest
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, err
	}

	return requests, nil
}

func (r *MongoRepository) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.client.Disconnect(ctx)
}
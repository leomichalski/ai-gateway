package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/leomichalski/aigateway/internal/domain"
)

// TODO: move to domain/errors.go or to domain/identity.go
var ErrIdentityNotFound = errors.New("identity not found")

type MongoIdentityRepository struct {
	collection *mongo.Collection
}

// NewMongoIdentityRepository initializes the repo with the specific database.
func NewMongoIdentityRepository(db *mongo.Database) *MongoIdentityRepository {
	return &MongoIdentityRepository{
		collection: db.Collection("identities"),
	}
}

func (r *MongoIdentityRepository) GetIdentityByAPIKey(ctx context.Context, keyHash string) (domain.Identity, error) {
	// Internal struct to handle BSON unmarshaling without polluting the Domain layer
	var doc struct {
		TenantID   string `bson:"tenant_id"`
		Tier       string `bson:"tier"`
		APIKeyHash string `bson:"api_key_hash"`
	}

	filter := bson.M{"api_key_hash": keyHash}
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Identity{}, ErrIdentityNotFound
		}
		// Return other DB connection errors
		return domain.Identity{}, err
	}

	// Map to standard Domain entity
	return domain.Identity{
		TenantID:   doc.TenantID,
		Tier:       doc.Tier,
		APIKeyHash: doc.APIKeyHash,
	}, nil
}
package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/renproject/multichain-proxy/pkg/shared"
	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	ConnectionTimeout = 10 * time.Second
	ProxyDB           = "proxy"
	ConfigCollection  = "config"
)

// Connect creates a new database client with the current configuration
func Connect(server, username, passwd string) (*mongo.Client, error) {
	credential := options.Credential{
		Username: username,
		Password: passwd,
	}

	clientOptions := options.Client().ApplyURI(server).SetAuth(credential)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db, error=%w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping db, error=%w", err)
	}

	return client, nil
}

type DBManager struct {
	Client           *mongo.Client
	ConfigCollection *mongo.Collection
}

func NewDBManager() (*DBManager, error) {
	server := os.Getenv("DB_SERVER")
	username := os.Getenv("DB_USER")
	passwd := os.Getenv("DB_PASSWORD")

	if server == "" || username == "" || passwd == "" {
		return nil, fmt.Errorf("missing database configuration")
	}
	client, err := Connect(server, username, passwd)
	if err != nil {
		return nil, fmt.Errorf("failed to create db client, error: %w", err)
	}
	configCol := client.Database(ProxyDB).Collection(ConfigCollection)
	_, err = configCol.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys: bson.M{
				"key": 1,
			},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create collection index, error: %w", err)
	}
	return &DBManager{
		Client:           client,
		ConfigCollection: configCol,
	}, nil
}

func (db *DBManager) CreateConfig(ctx context.Context, key string, value shared.ProxyConfig) error {
	_, err := db.ConfigCollection.InsertOne(ctx, shared.ProxyConfigDB{Key: key, Value: value})
	if err != nil {
		return fmt.Errorf("failed to insert entry, error: %w", err)
	}
	return nil
}

func (db *DBManager) UpdateConfig(ctx context.Context, key string, value shared.ProxyConfig) error {
	_, err := db.ConfigCollection.UpdateOne(ctx, bson.D{{"key", key}}, shared.ProxyConfigDB{Key: key, Value: value})
	if err != nil {
		return fmt.Errorf("failed to update entry, error: %w", err)
	}
	return nil
}

func (db *DBManager) GetConfig(ctx context.Context, key string) (*shared.ProxyConfig, error) {
	res := db.ConfigCollection.FindOne(ctx, bson.D{{"key", key}})
	var config shared.ProxyConfigDB
	err := res.Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get entry, error: %w", err)
	}
	return &config.Value, nil
}

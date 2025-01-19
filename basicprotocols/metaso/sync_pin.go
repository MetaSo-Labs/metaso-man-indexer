package metaso

import (
	"context"
	"fmt"
	"log"
	"manindexer/common"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (metaso *MetaSo) SyncPin(sysncSize int) {
	mgcfig := common.Config.MongoDb
	metasoConfig := common.Config.MetaSo
	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	// Connect to MongoDB for dbA and dbB
	clientOptions := options.Client().ApplyURI(metasoConfig.MongoNodeURI)
	clientA, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB for dbA: %v", err)
	}
	defer func() {
		if err = clientA.Disconnect(ctx); err != nil {
			log.Fatalf("Failed to disconnect MongoDB for dbA: %v", err)
		}
	}()

	// Access databases
	dbA := clientA.Database(mgcfig.DbName)
	collections := []string{"pins", "pintreee", "metaid"}
	for _, collection := range collections {
		err = copyCollection(dbA, mongoClient, collection, sysncSize)
		if err != nil {
			log.Fatalf("Failed to copy collection: %v", err)
		}
	}
	fmt.Println("Data copy complete!")
}

func copyCollection(dbA, dbB *mongo.Database, collectionName string, sysncSize int) error {
	ctx := context.Background()
	collectionA := dbA.Collection(collectionName)
	collectionB := dbB.Collection(collectionName)

	// Check if collectionB exists, if not, create it
	collections, err := dbB.ListCollectionNames(ctx, bson.M{"name": collectionName})
	if err != nil {
		return fmt.Errorf("failed to list collection names: %v", err)
	}
	if len(collections) == 0 {
		err := dbB.CreateCollection(ctx, collectionName)
		if err != nil {
			return fmt.Errorf("failed to create collection: %v", err)
		}
	}

	var lastID primitive.ObjectID

	batchSize := int64(sysncSize)
	for {
		var filter bson.M
		if !lastID.IsZero() {
			filter = bson.M{"_id": bson.M{"$gt": lastID}}
		}
		//fmt.Println(lastID)
		opts := options.Find().
			SetSort(bson.M{"_id": 1}).
			SetLimit(batchSize)

		cursor, err := collectionA.Find(ctx, filter, opts)
		if err != nil {
			return fmt.Errorf("failed to find documents: %v", err)
		}
		defer cursor.Close(ctx)

		var documents []interface{}
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				log.Printf("Failed to decode document: %v", err)
				continue
			}
			documents = append(documents, doc)
			// Update lastID to the _id of the last document
			if len(documents) > 0 {
				lastIDValue, exists := doc["_id"]
				if exists {
					if objectID, ok := lastIDValue.(primitive.ObjectID); ok {
						lastID = objectID
					} else {
						log.Printf("_id is not of type ObjectID: %v", lastIDValue)
					}
				}
			}
		}
		if err := cursor.Err(); err != nil {
			return fmt.Errorf("cursor error: %v", err)
		}

		if len(documents) == 0 {
			break // No more documents, exit loop
		}

		_, err = collectionB.InsertMany(ctx, documents, options.InsertMany().SetOrdered(false))
		if err != nil {
			if _, ok := err.(mongo.BulkWriteException); !ok {
				return fmt.Errorf("failed to insert documents: %v", err)
			}
		}
		log.Printf("%s Copied %d documents\n", collectionName, len(documents))
	}

	return nil
}

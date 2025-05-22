package metaso

import (
	"context"
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/database/mongodb"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetRecommendedPosts retrieves a list of recommended posts
// Including: 1. Posts marked as recommended 2. Posts from followed users
func (metaso *MetaSo) GetRecommendedPosts(ctx context.Context, lastId string, userAddress string, size int64) (listData []*TweetWithLike, total int64, err error) {
	userMetaId := ""
	var followedUsers []string
	if userAddress != "" {
		userMetaId = common.GetMetaIdByAddress(userAddress)
		// First, get the list of followed users
		followedUsers, err = metaso.getFollowedUsers(ctx, userMetaId)
		if err != nil {
			return
		}
	}

	// Build match conditions
	matchConditions := bson.D{{"isrecommended", true}}
	totalFilter := bson.D{{"isrecommended", true}}
	if len(followedUsers) > 0 {
		matchConditions = bson.D{
			{"$or", bson.A{
				bson.D{{"isrecommended", true}},
				bson.D{{"metaid", bson.D{{"$in", followedUsers}}}},
			}},
		}
		totalFilter = bson.D{
			{"$or", bson.A{
				bson.D{{"isrecommended", true}},
				bson.D{{"metaid", bson.D{{"$in", followedUsers}}}},
			}},
		}
	}
	// Add lastId condition if provided
	if lastId != "" {
		var lastObjectID primitive.ObjectID
		lastObjectID, err = primitive.ObjectIDFromHex(lastId)
		if err != nil {
			return
		}
		matchConditions = append(matchConditions, bson.E{
			"_id", bson.D{{"$lt", lastObjectID}},
		})
	}

	// Build find options
	findOptions := options.Find().
		SetSort(bson.D{{"_id", -1}}).
		SetLimit(size)

	// Execute query
	cursor, err := mongoClient.Collection(BuzzView).Find(ctx, matchConditions, findOptions)
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	// Parse results
	var list []*Tweet
	err = cursor.All(context.TODO(), &list)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	var pinIdList []string
	for _, item := range list {
		item.Content = string(item.ContentBody)
		item.ContentBody = nil
		pinIdList = append(pinIdList, item.Id)
	}

	mempoolList, err := getBuzzMempoolCount(pinIdList)
	if err == nil {
		for _, item := range list {
			for _, data := range mempoolList {
				if item.Id == data.Target && data.Path == "/protocols/paylike" {
					item.LikeCount += 1
				}
				if item.Id == data.Target && data.Path == "/protocols/paycomment" {
					item.CommentCount += 1
				}
				if item.Id == data.Target && data.Path == "/protocols/simpledonate" {
					item.DonateCount += 1
				}
			}
		}
	}
	checkMap := make(map[string]*TweetWithLike, len(list))
	for _, item := range list {
		checkMap[item.Id] = &TweetWithLike{Tweet: *item, Like: []string{}, Donate: []string{}}
	}
	likeMap, err := batchGetPayLike(pinIdList)
	if err == nil {
		for _, item := range list {
			if v, ok := likeMap[item.Id]; ok {
				checkMap[item.Id].Like = v
			}
		}
	}
	donateMap, err := batchGetSimpleDonat(pinIdList)
	if err == nil {
		for _, item := range list {
			if v, ok := donateMap[item.Id]; ok {
				checkMap[item.Id].Donate = v
			}
		}
	}
	for _, item := range list {
		if v, ok := checkMap[item.Id]; ok {
			listData = append(listData, v)
		}
	}
	total, err = mongoClient.Collection(BuzzView).CountDocuments(context.TODO(), totalFilter)

	return
}

// getFollowedUsers gets the list of users that the current user follows
func (metaso *MetaSo) getFollowedUsers(ctx context.Context, userMetaID string) ([]string, error) {
	// Find all follows where the current user is the follower
	cursor, err := mongoClient.Collection(mongodb.FollowCollection).Find(ctx, bson.D{
		{"followmetaid", userMetaID}, {"status", true},
	}, options.Find().SetProjection(bson.D{{"metaid", 1}, {"_id", 0}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var follows []struct {
		MetaID string `bson:"metaid"`
	}
	if err = cursor.All(ctx, &follows); err != nil {
		return nil, err
	}

	// Extract metaids
	metaids := make([]string, len(follows))
	for i, follow := range follows {
		metaids[i] = follow.MetaID
	}

	return metaids, nil
}

// AddRecommendedAuthor adds an author to the recommended authors list
func (metaso *MetaSo) AddRecommendedAuthor(ctx context.Context, authorID string, authorName string) error {
	// Check if author is already recommended
	filter := bson.D{{"author_id", authorID}}
	var existing RecommendedAuthor
	err := mongoClient.Collection(RecommendedAuthors).FindOne(ctx, filter).Decode(&existing)
	if err == nil {
		return nil // Author is already recommended
	} else if err != mongo.ErrNoDocuments {
		return err // Other error occurred
	}

	// Create new recommended author record
	now := time.Now()
	recommendedAuthor := RecommendedAuthor{
		AuthorID:   authorID,
		AuthorName: authorName,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Insert recommended author record
	_, err = mongoClient.Collection(RecommendedAuthors).InsertOne(ctx, recommendedAuthor)
	if err != nil {
		return err
	}

	// Asynchronously update all posts by this author to recommended
	go metaso.asyncUpdateAuthorPostsRecommendation(authorID, true)
	common.RecommendedAuthor[authorID] = struct{}{}
	for k, _ := range common.RecommendedAuthor {
		fmt.Println(k)
	}
	return nil
}

// RemoveRecommendedAuthor removes an author from the recommended authors list
func (metaso *MetaSo) RemoveRecommendedAuthor(ctx context.Context, authorID string) error {
	// Delete recommended author record
	filter := bson.D{{"author_id", authorID}}
	result, err := mongoClient.Collection(RecommendedAuthors).DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	// Asynchronously update all posts by this author to non-recommended
	go metaso.asyncUpdateAuthorPostsRecommendation(authorID, false)
	if _, exists := common.RecommendedAuthor[authorID]; exists {
		delete(common.RecommendedAuthor, authorID)
	}
	return nil
}

// asyncUpdateAuthorPostsRecommendation asynchronously updates the recommendation status of all posts by an author
func (metaso *MetaSo) asyncUpdateAuthorPostsRecommendation(authorID string, isRecommended bool) {
	const batchSize = 1000 // Number of posts to process in each batch
	var wg sync.WaitGroup
	ctx := context.Background()

	// Create new context to avoid using cancelled context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Process in batches
	for {
		// Find a batch of posts to update
		filter := bson.D{{"address", authorID}}
		opts := options.Find().SetProjection(bson.M{"_id": 1}).SetLimit(batchSize)
		cursor, err := mongoClient.Collection(TweetCollection).Find(ctx, filter, opts)
		defer cursor.Close(context.TODO())
		if err != nil {
			log.Printf("Failed to find posts: %v", err)
			return
		}
		// Get IDs of this batch of posts
		var postIDs []primitive.ObjectID
		for cursor.Next(context.TODO()) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				return
			}
			if id, ok := doc["_id"].(primitive.ObjectID); ok {
				postIDs = append(postIDs, id)
			}
		}
		// Asynchronously update this batch of posts
		wg.Add(1)
		go func(postIDs []primitive.ObjectID) {
			defer wg.Done()
			updateFilter := bson.D{{"_id", bson.D{{"$in", postIDs}}}}
			update := bson.D{{"$set", bson.D{{"isrecommended", isRecommended}}}}

			_, err := mongoClient.Collection(TweetCollection).UpdateMany(ctx, updateFilter, update)
			if err != nil {
				log.Printf("Failed to update post recommendation status: %v", err)
			}
		}(postIDs)

		// If this batch is smaller than batch size, we've processed all posts
		if len(postIDs) < batchSize {
			break
		}
	}

	// Wait for all updates to complete
	wg.Wait()
	log.Printf("Completed updating recommendation status for all posts by author %s", authorID)
}

// GetRecommendedAuthors retrieves a paginated list of recommended authors
func (metaso *MetaSo) GetRecommendedAuthors(ctx context.Context, findCursor, pageSize int64) ([]RecommendedAuthor, int64, error) {

	// Set up find options for pagination
	findOptions := options.Find().
		SetSkip(findCursor).
		SetLimit(pageSize).
		SetSort(bson.D{{"created_at", -1}}) // Sort by creation time in descending order

	// Get total count of recommended authors
	total, err := mongoClient.Collection(RecommendedAuthors).CountDocuments(ctx, bson.D{})
	if err != nil {
		return nil, 0, err
	}

	// Find recommended authors with pagination
	cursor, err := mongoClient.Collection(RecommendedAuthors).Find(ctx, bson.D{}, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var authors []RecommendedAuthor
	if err = cursor.All(ctx, &authors); err != nil {
		return nil, 0, err
	}

	return authors, total, nil
}

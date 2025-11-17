package repository

import (
    "context"
    "errors"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("not found")

type Chat struct {
    ID         string    `bson:"_id,omitempty" json:"id"`
    Name       string    `bson:"name,omitempty" json:"name"`
    IsGroup    bool      `bson:"is_group" json:"is_group"`
    Members    []string  `bson:"members" json:"members"`
    CreatedAt  time.Time `bson:"created_at" json:"created_at"`
    UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
}

type Repository struct {
    coll *mongo.Collection
}

func NewMongoRepository(coll *mongo.Collection) *Repository {
    // ensure index on members for listing
    idx := mongo.IndexModel{
        Keys:    bson.D{{Key: "members", Value: 1}},
        Options: options.Index().SetBackground(true).SetName("members_idx"),
    }
    _, _ = coll.Indexes().CreateOne(context.Background(), idx)
    return &Repository{coll: coll}
}

func (r *Repository) CreateChat(ctx context.Context, chat *Chat) error {
    now := time.Now().UTC()
    chat.CreatedAt = now
    chat.UpdatedAt = now
    _, err := r.coll.InsertOne(ctx, chat)
    return err
}

func (r *Repository) GetChat(ctx context.Context, id string) (*Chat, error) {
    var c Chat
    if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&c); err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, ErrNotFound
        }
        return nil, err
    }
    return &c, nil
}

func (r *Repository) ListChatsForUser(ctx context.Context, userID string, limit int64) ([]*Chat, error) {
    cur, err := r.coll.Find(ctx, bson.M{"members": userID}, &options.FindOptions{
        Sort:  bson.D{{Key: "updated_at", Value: -1}},
        Limit: &limit,
    })
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)
    var out []*Chat
    for cur.Next(ctx) {
        var c Chat
        if err := cur.Decode(&c); err != nil {
            return nil, err
        }
        out = append(out, &c)
    }
    return out, nil
}

func (r *Repository) AddMember(ctx context.Context, chatID, userID string) error {
    _, err := r.coll.UpdateByID(ctx, chatID, bson.M{"$addToSet": bson.M{"members": userID}, "$set": bson.M{"updated_at": time.Now().UTC()}})
    return err
}

func (r *Repository) RemoveMember(ctx context.Context, chatID, userID string) error {
    _, err := r.coll.UpdateByID(ctx, chatID, bson.M{"$pull": bson.M{"members": userID}, "$set": bson.M{"updated_at": time.Now().UTC()}})
    return err
}

func (r *Repository) UpdateChatName(ctx context.Context, chatID, name string) error {
    _, err := r.coll.UpdateByID(ctx, chatID, bson.M{"$set": bson.M{"name": name, "updated_at": time.Now().UTC()}})
    return err
}

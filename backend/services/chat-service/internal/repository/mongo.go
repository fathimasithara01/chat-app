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

type Member struct {
    ID       string `bson:"id" json:"id"`
    Username string `bson:"username" json:"username"`
    Avatar   string `bson:"avatar,omitempty" json:"avatar,omitempty"`
}

type Message struct {
    ID        string    `bson:"_id" json:"id"`
    Sender    Member    `bson:"sender" json:"sender"`
    Content   string    `bson:"content" json:"content"`
    MsgType   string    `bson:"msg_type" json:"msg_type"`
    CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type Chat struct {
    ID          string    `bson:"_id,omitempty" json:"id"`
    Name        string    `bson:"name,omitempty" json:"name"`
    IsGroup     bool      `bson:"is_group" json:"is_group"`
    Members     []Member  `bson:"members" json:"members"`
    LastMessage *Message  `bson:"last_message,omitempty" json:"last_message,omitempty"`
    CreatedAt   time.Time `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

type Repository struct {
    coll *mongo.Collection
}

func NewMongoRepository(coll *mongo.Collection) *Repository {
    idx := mongo.IndexModel{
        Keys:    bson.D{{Key: "members.id", Value: 1}},
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
    cur, err := r.coll.Find(ctx, bson.M{"members.id": userID}, &options.FindOptions{
        Sort:  bson.D{{Key: "updated_at", Value: -1}},
        Limit: &limit,
    })
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)

    var chats []*Chat
    for cur.Next(ctx) {
        var c Chat
        if err := cur.Decode(&c); err != nil {
            return nil, err
        }
        chats = append(chats, &c)
    }
    return chats, nil
}

func (r *Repository) AddMember(ctx context.Context, chatID string, member Member) error {
    update := bson.M{
        "$addToSet": bson.M{"members": member},
        "$set":      bson.M{"updated_at": time.Now().UTC()},
    }
    _, err := r.coll.UpdateByID(ctx, chatID, update)
    return err
}

func (r *Repository) RemoveMember(ctx context.Context, chatID, memberID string) error {
    update := bson.M{
        "$pull": bson.M{"members": bson.M{"id": memberID}},
        "$set":  bson.M{"updated_at": time.Now().UTC()},
    }
    _, err := r.coll.UpdateByID(ctx, chatID, update)
    return err
}

func (r *Repository) UpdateChatName(ctx context.Context, chatID, name string) error {
    update := bson.M{
        "$set": bson.M{"name": name, "updated_at": time.Now().UTC()},
    }
    _, err := r.coll.UpdateByID(ctx, chatID, update)
    return err
}

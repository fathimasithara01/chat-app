package domain

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

type User struct {
	ID       string   `json:"id" bson:"_id,omitempty"`
	Name     string   `json:"name" bson:"name"`
	Email    string   `json:"email" bson:"email"`
	Phone    string   `json:"phone,omitempty" bson:"phone,omitempty"`
	Password string   `json:"-" bson:"password"`
	Role     UserRole `json:"role" bson:"role"`
	Verified bool     `json:"verified" bson:"verified"`
}

type UserRegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone,omitempty"`
	Password string `json:"password"`
}

type UserUpdateRequest struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

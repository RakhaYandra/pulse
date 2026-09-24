package usecase

import (
	"context"
	"strings"

	"github.com/RakhaYandra/pulse/domain"
	"github.com/google/uuid"
)

type UserDTO struct {
	ID    string
	Email string
	Name  string
}

type AuthOutput struct {
	Token string
	User  UserDTO
}

type AuthService struct {
	Users  UserRepo
	Hash   Hasher
	Tokens TokenIssuer
}

func toUserDTO(u domain.User) UserDTO {
	return UserDTO{ID: u.ID, Email: u.Email, Name: u.Name}
}

func (s AuthService) Register(ctx context.Context, email, password, name string) (AuthOutput, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	u := domain.User{ID: uuid.NewString(), Email: email, Name: name}
	if err := u.ValidateNew(password); err != nil {
		return AuthOutput{}, err
	}
	if !strings.Contains(email, "@") {
		return AuthOutput{}, &domain.FieldError{Field: "email", Message: "email invalid"}
	}
	hash, err := s.Hash.Hash(password)
	if err != nil {
		return AuthOutput{}, err
	}
	u.PasswordHash = hash
	if err := s.Users.Create(ctx, u); err != nil {
		return AuthOutput{}, err
	}
	tok, err := s.Tokens.Issue(u.ID)
	if err != nil {
		return AuthOutput{}, err
	}
	return AuthOutput{Token: tok, User: toUserDTO(u)}, nil
}

func (s AuthService) Login(ctx context.Context, email, password string) (AuthOutput, error) {
	u, err := s.Users.ByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		return AuthOutput{}, domain.ErrUnauthorized
	}
	if err := s.Hash.Compare(u.PasswordHash, password); err != nil {
		return AuthOutput{}, domain.ErrUnauthorized
	}
	tok, err := s.Tokens.Issue(u.ID)
	if err != nil {
		return AuthOutput{}, err
	}
	return AuthOutput{Token: tok, User: toUserDTO(u)}, nil
}

func (s AuthService) Me(ctx context.Context, userID string) (UserDTO, error) {
	u, err := s.Users.ByID(ctx, userID)
	if err != nil {
		return UserDTO{}, err
	}
	return toUserDTO(u), nil
}

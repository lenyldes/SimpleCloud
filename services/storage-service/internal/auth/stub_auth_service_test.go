package auth_test

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
)

// stubAuthService is a test double for auth.Service. Users, passwords and
// sessions are supplied explicitly by each test; it contains no hardcoded
// production credentials (BUGS.md C1).
type stubAuthService struct {
	mu       sync.Mutex
	users    map[uuid.UUID]*stubUser
	byEmail  map[string]uuid.UUID
	sessions map[string]stubSession
}

type stubUser struct {
	user     auth.User
	password string
}

type stubSession struct {
	userID    uuid.UUID
	expiresAt time.Time
}

func newStubAuthService() *stubAuthService {
	return &stubAuthService{
		users:    make(map[uuid.UUID]*stubUser),
		byEmail:  make(map[string]uuid.UUID),
		sessions: make(map[string]stubSession),
	}
}

func (s *stubAuthService) AddUser(user auth.User, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.ID] = &stubUser{user: user, password: password}
	s.byEmail[user.Email] = user.ID
}

// IssueSession registers a valid session token for userID; a user record is
// auto-created when absent so middleware tests stay self-contained.
func (s *stubAuthService) IssueSession(userID uuid.UUID, duration time.Duration) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		auto := auth.User{ID: userID, Email: userID.String() + "@stub.test", Role: "user"}
		s.users[userID] = &stubUser{user: auto}
		s.byEmail[auto.Email] = userID
	}
	token, _, err := auth.GenerateSessionToken()
	if err != nil {
		panic(err)
	}
	s.sessions[token] = stubSession{userID: userID, expiresAt: time.Now().Add(duration)}
	return token
}

func (s *stubAuthService) Login(_ context.Context, email, password, _, _ string) (string, *auth.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.byEmail[email]
	if !ok {
		return "", nil, auth.ErrInvalidCredentials
	}
	su := s.users[id]
	if su.password == "" || su.password != password {
		return "", nil, auth.ErrInvalidCredentials
	}

	token, _, err := auth.GenerateSessionToken()
	if err != nil {
		return "", nil, err
	}
	s.sessions[token] = stubSession{userID: id, expiresAt: time.Now().Add(time.Hour)}
	user := su.user
	return token, &user, nil
}

func (s *stubAuthService) Logout(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	return nil
}

func (s *stubAuthService) ValidateSession(_ context.Context, token string) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[token]
	if !ok || time.Now().After(sess.expiresAt) {
		return uuid.Nil, auth.ErrUnauthorized
	}
	return sess.userID, nil
}

func (s *stubAuthService) GetUserByID(_ context.Context, userID uuid.UUID) (*auth.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	su, ok := s.users[userID]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	user := su.user
	return &user, nil
}

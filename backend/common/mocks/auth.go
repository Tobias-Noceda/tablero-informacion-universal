package mocks

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// MemoryUserStore is a UserStore over a slice, enough for a service test.
type MemoryUserStore struct {
	Users []models.User
}

var _ infrastructure.UserStore = (*MemoryUserStore)(nil)

func (m *MemoryUserStore) CreateUser(user *models.User) error {
	for _, existing := range m.Users {
		if existing.Email == user.Email {
			return infrastructure.ErrEmailTaken
		}
	}
	m.Users = append(m.Users, *user)
	return nil
}

func (m *MemoryUserStore) FindUser(id uuid.UUID) (*models.User, error) {
	return m.find(func(u *models.User) bool { return u.Id == id })
}

func (m *MemoryUserStore) FindUserByEmail(email string) (*models.User, error) {
	return m.find(func(u *models.User) bool { return u.Email == email })
}

func (m *MemoryUserStore) FindUserByIdentity(provider models.IdentityProvider, subject string) (*models.User, error) {
	return m.find(func(u *models.User) bool {
		identity, ok := u.Identity(provider)
		return ok && identity.Subject == subject
	})
}

func (m *MemoryUserStore) FindUsers(ids []string) ([]models.User, error) {
	var out []models.User
	for _, user := range m.Users {
		if slices.Contains(ids, user.Id.String()) {
			out = append(out, user)
		}
	}
	return out, nil
}

func (m *MemoryUserStore) UpdateUser(id uuid.UUID, set map[string]any) error {
	user := m.at(id)
	if user == nil {
		return infrastructure.ErrUserNotFound
	}
	for field, value := range set {
		switch field {
		case "name":
			user.Name = value.(string)
		case "picture":
			user.Picture = value.(string)
		case "admin":
			user.Admin = value.(bool)
		case "emailverified":
			user.EmailVerified = value.(bool)
		}
	}
	user.UpdatedAt = time.Now()
	return nil
}

func (m *MemoryUserStore) ReplaceIdentities(id uuid.UUID, identities []models.Identity) error {
	user := m.at(id)
	if user == nil {
		return infrastructure.ErrUserNotFound
	}
	user.Identities = slices.Clone(identities)
	return nil
}

func (m *MemoryUserStore) find(match func(*models.User) bool) (*models.User, error) {
	for i := range m.Users {
		if match(&m.Users[i]) {
			user := m.Users[i]
			user.Identities = slices.Clone(m.Users[i].Identities)
			return &user, nil
		}
	}
	return nil, infrastructure.ErrUserNotFound
}

func (m *MemoryUserStore) at(id uuid.UUID) *models.User {
	for i := range m.Users {
		if m.Users[i].Id == id {
			return &m.Users[i]
		}
	}
	return nil
}

// MemorySessionStore keeps refresh tokens, spent markers and families in maps.
type MemorySessionStore struct {
	mu       sync.Mutex
	sessions map[string]infrastructure.Session
	spent    map[string]spentMark
	families map[string][]string
	users    map[string][]string
}

type spentMark struct {
	family string
	at     time.Time
}

var _ infrastructure.SessionStore = (*MemorySessionStore)(nil)

func (m *MemorySessionStore) init() {
	if m.sessions == nil {
		m.sessions = map[string]infrastructure.Session{}
		m.spent = map[string]spentMark{}
		m.families = map[string][]string{}
		m.users = map[string][]string{}
	}
}

func (m *MemorySessionStore) PutSession(session infrastructure.Session, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	m.sessions[session.TokenHash] = session
	m.families[session.Family] = append(m.families[session.Family], session.TokenHash)
	if !slices.Contains(m.users[session.UserID], session.Family) {
		m.users[session.UserID] = append(m.users[session.UserID], session.Family)
	}
	return nil
}

func (m *MemorySessionStore) TakeSession(tokenHash string) (*infrastructure.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	session, ok := m.sessions[tokenHash]
	if !ok {
		return nil, infrastructure.ErrSessionNotFound
	}
	delete(m.sessions, tokenHash)
	return &session, nil
}

func (m *MemorySessionStore) MarkSpent(tokenHash, family string, at time.Time, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	m.spent[tokenHash] = spentMark{family: family, at: at}
	return nil
}

func (m *MemorySessionStore) Spent(tokenHash string) (string, time.Time, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	mark, ok := m.spent[tokenHash]
	return mark.family, mark.at, ok, nil
}

func (m *MemorySessionStore) RevokeFamily(family string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	for _, hash := range m.families[family] {
		delete(m.sessions, hash)
	}
	delete(m.families, family)
	return nil
}

func (m *MemorySessionStore) RevokeUser(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	for _, family := range m.users[userID] {
		for _, hash := range m.families[family] {
			delete(m.sessions, hash)
		}
		delete(m.families, family)
	}
	delete(m.users, userID)
	return nil
}

// Live returns how many refresh tokens are still redeemable.
func (m *MemorySessionStore) Live() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	return len(m.sessions)
}

// StaticVerifier accepts exactly the tokens it was given.
type StaticVerifier map[string]infrastructure.Claims

var _ infrastructure.TokenVerifier = StaticVerifier{}

func (v StaticVerifier) Verify(token string) (infrastructure.Claims, error) {
	claims, ok := v[token]
	if !ok {
		return infrastructure.Claims{}, infrastructure.ErrInvalidToken
	}
	return claims, nil
}

// PlainHasher stores passwords in the clear so service tests stay fast.
type PlainHasher struct{}

var _ infrastructure.PasswordHasher = PlainHasher{}

func (PlainHasher) Hash(password string) (string, error) { return "plain:" + password, nil }

func (PlainHasher) Compare(encoded, password string) (bool, error) {
	return encoded == "plain:"+password, nil
}

// RecordingMailer captures every mail so a test can read the links out of it.
type RecordingMailer struct {
	mu    sync.Mutex
	Mails []models.Mail
}

var _ infrastructure.Mailer = (*RecordingMailer)(nil)

func (m *RecordingMailer) Send(mail models.Mail) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Mails = append(m.Mails, mail)
	return nil
}

func (m *RecordingMailer) Last() (models.Mail, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Mails) == 0 {
		return models.Mail{}, false
	}
	return m.Mails[len(m.Mails)-1], true
}

// CountingLimiter allows every call unless a bucket was told to refuse.
type CountingLimiter struct {
	mu     sync.Mutex
	Calls  map[string]int
	Refuse map[string]bool
}

var _ infrastructure.RateLimiter = (*CountingLimiter)(nil)

func (l *CountingLimiter) Allow(key string, limit int, window time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.Calls == nil {
		l.Calls = map[string]int{}
	}
	l.Calls[key]++
	return !l.Refuse[key], nil
}

// MockIdentityProvider returns whatever identity the test configured.
type MockIdentityProvider struct {
	Identity *infrastructure.ExternalIdentity
	Err      error
	Seen     map[string]string
}

var _ infrastructure.IdentityProvider = (*MockIdentityProvider)(nil)

func (p *MockIdentityProvider) AuthURL(state, nonce, codeChallenge, redirectURI string) string {
	return "https://idp.test/authorize?state=" + state + "&nonce=" + nonce + "&code_challenge=" + codeChallenge + "&redirect_uri=" + redirectURI
}

func (p *MockIdentityProvider) Complete(code, redirectURI, verifier, nonce string) (*infrastructure.ExternalIdentity, error) {
	if p.Seen == nil {
		p.Seen = map[string]string{}
	}
	maps.Copy(p.Seen, map[string]string{"code": code, "redirect_uri": redirectURI, "verifier": verifier, "nonce": nonce})
	if p.Err != nil {
		return nil, p.Err
	}
	identity := *p.Identity
	return &identity, nil
}

package cache

import (
	"sync"

	"github.com/denchenko/gg/internal/core/domain"
)

// InMemoryCache is an in-memory thread-safe cache implementation for users.
type InMemoryCache struct {
	byID       sync.Map // map[int]*domain.User
	byUsername sync.Map // map[string]*domain.User
}

// NewInMemoryCache creates a new in-memory cache instance.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		byID:       sync.Map{},
		byUsername: sync.Map{},
	}
}

// GetUserByID retrieves a user by ID from the cache.
func (c *InMemoryCache) GetUserByID(id int) (*domain.User, bool) {
	if cached, ok := c.byID.Load(id); ok {
		if user, ok := cached.(*domain.User); ok {
			return user, true
		}
	}

	return nil, false
}

// GetUserByUsername retrieves a user by username from the cache.
func (c *InMemoryCache) GetUserByUsername(username string) (*domain.User, bool) {
	if cached, ok := c.byUsername.Load(username); ok {
		if user, ok := cached.(*domain.User); ok {
			return user, true
		}
	}

	return nil, false
}

// StoreUser stores a user in the cache, indexed by both ID and username.
func (c *InMemoryCache) StoreUser(user *domain.User) {
	c.byID.Store(user.ID, user)
	c.byUsername.Store(user.Username, user)
}

// GetAllUsers retrieves all users from the cache.
func (c *InMemoryCache) GetAllUsers() []*domain.User {
	var users []*domain.User

	c.byID.Range(func(_ any, value any) bool {
		if user, ok := value.(*domain.User); ok {
			users = append(users, user)
		}

		return true
	})

	return users
}

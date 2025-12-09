package cache

import "github.com/denchenko/gg/internal/core/domain"

// Cache defines the interface for user caching operations.
type Cache interface {
	// GetUserByID retrieves a user by ID from the cache.
	// Returns the user and true if found, nil and false otherwise.
	GetUserByID(id int) (*domain.User, bool)

	// GetUserByUsername retrieves a user by username from the cache.
	// Returns the user and true if found, nil and false otherwise.
	GetUserByUsername(username string) (*domain.User, bool)

	// StoreUser stores a user in the cache, indexed by both ID and username.
	StoreUser(user *domain.User)

	// GetAllUsers retrieves all users from the cache.
	GetAllUsers() []*domain.User
}

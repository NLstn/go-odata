package entities

// User represents a user entity
//
// ⚠️  SECURITY WARNING: This is an incomplete example!
//
// This User entity is missing critical security features required for production:
//
// 1. NO PASSWORD FIELD: Cannot authenticate users
// 2. NO PASSWORD HASHING: Passwords must never be stored in plaintext
// 3. INCOMPLETE ROLE MANAGEMENT: IsAdmin is boolean, real systems need RBAC
//
// For production, you must add:
// - PasswordHash field (with json:"-" to exclude from responses)
// - Password hashing using bcrypt or argon2
// - Proper role/permission management
// - Account security fields (locked, failed attempts, etc.)
//
// Example secure implementation:
//
//	type User struct {
//	    UserID       uint      `json:"UserID" gorm:"primaryKey" odata:"key"`
//	    Name         string    `json:"Name" gorm:"not null" odata:"required"`
//	    Email        string    `json:"Email" gorm:"uniqueIndex;not null"`
//	    PasswordHash string    `json:"-" gorm:"not null"` // Never expose!
//	    IsAdmin      bool      `json:"IsAdmin" gorm:"default:false"`
//	    IsLocked     bool      `json:"-" gorm:"default:false"`
//	    LastLogin    time.Time `json:"LastLogin"`
//	}
//
// See SECURITY.md for complete secure user management examples.
type User struct {
	UserID  uint   `json:"UserID" gorm:"primaryKey" odata:"key"`
	Name    string `json:"Name" gorm:"not null" odata:"required,maxlength=100,searchable"`
	IsAdmin bool   `json:"IsAdmin" gorm:"not null;default:false"`
}

// GetSampleUsers returns sample user data for seeding the database
func GetSampleUsers() []User {
	return []User{
		{
			UserID:  1,
			Name:    "Alice Johnson",
			IsAdmin: true,
		},
		{
			UserID:  2,
			Name:    "Bob Smith",
			IsAdmin: false,
		},
		{
			UserID:  3,
			Name:    "Charlie Davis",
			IsAdmin: true,
		},
		{
			UserID:  4,
			Name:    "Diana Martinez",
			IsAdmin: false,
		},
		{
			UserID:  5,
			Name:    "Eve Wilson",
			IsAdmin: false,
		},
	}
}

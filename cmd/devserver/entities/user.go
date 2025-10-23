package entities

// User represents a user entity
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

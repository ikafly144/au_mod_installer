package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_Creation(t *testing.T) {
	u := User{
		ID:       1,
		Username: "admin",
		IsAdmin:  true,
	}
	assert.Equal(t, "admin", u.Username)
	assert.True(t, u.IsAdmin)
}

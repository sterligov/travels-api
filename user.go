package main

import (
	"github.com/gin-gonic/gin"
)

// todo: можно обойтись и без структур
type User struct {
	ID        uint32 `json:"id"`
	Email     string `json:"email" `
	FirstName string `json:"first_name" db:"first_name"`
	LastName  string `json:"last_name" db:"last_name"`
	Gender    string `json:"gender"`
	BirthDate int    `json:"birth_date" db:"birth_date"`
}

func (a *api) retrieveUser(c *gin.Context) {
	a.retriveEntity("users", &User{}, c)
}

func (a *api) insertUser(c *gin.Context) {
	rules := map[string]string{
		"id":         "required",
		"email":      "required,email",
		"first_name": "required",
		"last_name":  "required",
		"gender":     "required,oneof=m f",
		"birth_date": "required",
	}
	a.insertEntity("users", rules, c)
}

func (a *api) updateUser(c *gin.Context) {
	rules := map[string]string{
		"email":  "email",
		"gender": "in=m f",
	}
	a.updateEntity("users", rules, c)
}

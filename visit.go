package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Visit struct {
	ID         uint32 `json:"id"`
	LocationID uint   `json:"location" db:"location"`
	UserID     uint   `json:"user" db:"user"`
	VisitedAt  int    `json:"visited_at" db:"visited_at"`
	Mark       int    `json:"mark"`
}

var userVisitsConds = queryParams{
	"toDistance": map[string]string{
		"type":      "int",
		"condition": "l.distance<?",
	},
	"country": map[string]string{
		"condition": "l.country=?",
	},
	"fromDate": map[string]string{
		"type":      "int",
		"condition": "v.visited_at>?",
	},
	"toDate": map[string]string{
		"type":      "int",
		"condition": "v.visited_at<?",
	},
}

func (a *api) retrieveVisit(c *gin.Context) {
	a.retriveEntity("visits", &Visit{}, c)
}

func (a *api) insertVisit(c *gin.Context) {
	rules := map[string]string{
		"id":         "required",
		"location":   "required",
		"user":       "required",
		"visited_at": "required",
		"mark":       "required,in=0 1 2 3 4 5",
	}
	a.insertEntity("visits", rules, c)
}

func (a *api) updateVisit(c *gin.Context) {
	rules := map[string]string{
		"mark": "in=0 1 2 3 4 5",
	}
	a.updateEntity("visits", rules, c)
}

func (a *api) retreiveUserVisits(c *gin.Context) {
	m := make(map[string]interface{})
	if err := a.getEntityByID(c.Param("id"), "users", m); err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	vals, cols, err := buildCondition(userVisitsConds, c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	vals = append(vals, c.Param("id"))
	cols = append(cols, "v.user=?")

	condition := strings.Join(cols, " AND ")
	query := fmt.Sprintf(`
		SELECT
			v.mark,
			v.visited_at, 
			l.place
		FROM
			visits v
			JOIN locations l ON v.location = l.id
		WHERE 
			%s
		ORDER BY 
			visited_at
	`, condition)

	visits := make([]map[string]interface{}, 0, 16)
	rows, err := a.storage.Queryx(query, vals...)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	for rows.Next() {
		m := make(map[string]interface{})
		if err := rows.MapScan(m); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		m["place"] = string(m["place"].([]byte))
		visits = append(visits, m)
	}

	c.JSON(http.StatusOK, gin.H{"visits": visits})
}

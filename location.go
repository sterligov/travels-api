package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Location struct {
	ID       uint32 `json:"id"`
	Place    string `json:"place"`
	Country  string `json:"country"`
	City     string `json:"city"`
	Distance int    `json:"distance"`
}

var locationsAvgConds = queryParams{
	"gender": map[string]string{
		"condition": "u.gender=?",
	},
	"fromDate": map[string]string{
		"type":      "int",
		"condition": "v.visited_at>?",
	},
	"toDate": map[string]string{
		"type":      "int",
		"condition": "v.visited_at<?",
	},
	"fromAge": map[string]string{
		"type":      "int",
		"condition": "u.birth_date<UNIX_TIMESTAMP() - TIMESTAMPDIFF(SECOND, NOW() - INTERVAL ? YEAR, NOW())",
	},
	"toAge": map[string]string{
		"type":      "int",
		"condition": "u.birth_date>UNIX_TIMESTAMP() - TIMESTAMPDIFF(SECOND, NOW() - INTERVAL ? YEAR, NOW())",
	},
}

func (a *api) retrieveLocation(c *gin.Context) {
	a.retriveEntity("locations", &Location{}, c)
}

func (a *api) updateLocation(c *gin.Context) {
	a.updateEntity("locations", map[string]string{}, c)
}

func (a *api) insertLocation(c *gin.Context) {
	rules := map[string]string{
		"id":       "required",
		"place":    "required",
		"country":  "required",
		"city":     "required",
		"distance": "required",
	}
	a.insertEntity("locations", rules, c)
}

func (a *api) retreiveAvg(c *gin.Context) {
	m := make(map[string]interface{})
	if err := a.getEntityByID(c.Param("id"), "locations", m); err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	gender, exist := c.GetQuery("gender")
	if err := a.validator.Var(gender, "oneof=m f"); exist && err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	vals, cols, err := buildCondition(locationsAvgConds, c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	vals = append(vals, c.Param("id"))
	cols = append(cols, "v.location=?")

	query := fmt.Sprintf(`
		SELECT
			ROUND(AVG(mark), 5) as avg
		FROM
			visits v
			JOIN users u ON u.id = v.user
		WHERE
			%s
	`, strings.Join(cols, " AND "))
	var avg float64
	a.storage.Get(&avg, query, vals...)

	c.JSON(http.StatusOK, gin.H{"avg": avg})
}

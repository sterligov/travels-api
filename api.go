package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/jmoiron/sqlx"
)

type queryParams map[string]map[string]string

type api struct {
	storage   *sqlx.DB
	validator *validator.Validate
}

func (a *api) Run() {
	gin.SetMode(os.Getenv("GIN_MODE"))
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Connection", "keep-alive")
	})

	r.GET("/users/:id", a.retrieveUser)
	r.GET("/users/:id/visits", a.retreiveUserVisits)
	r.GET("/locations/:id", a.retrieveLocation)
	r.GET("/visits/:id", a.retrieveVisit)
	r.GET("/locations/:id/avg", a.retreiveAvg)

	r.POST("/users/:id", func(c *gin.Context) {
		if c.Param("id") == "new" {
			a.insertUser(c)
		} else {
			a.updateUser(c)
		}
	})
	r.POST("/locations/:id", func(c *gin.Context) {
		if c.Param("id") == "new" {
			a.insertLocation(c)
		} else {
			a.updateLocation(c)
		}
	})
	r.POST("/visits/:id", func(c *gin.Context) {
		if c.Param("id") == "new" {
			a.insertVisit(c)
		} else {
			a.updateVisit(c)
		}
	})

	r.Run(":" + os.Getenv("APP_PORT"))
}

func (a *api) retriveEntity(table string, entity interface{}, c *gin.Context) {
	err := a.getEntityByID(c.Param("id"), table, entity)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (a *api) getEntityByID(id interface{}, table string, dst interface{}) error {
	query := fmt.Sprintf(`SELECT * FROM %s WHERE id = ?`, table)

	switch dst.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{})
		row := a.storage.QueryRowx(query, id)
		err := row.MapScan(m)
		if err != nil {
			return err
		}
		for k, v := range m {
			switch v.(type) {
			case []byte:
				m[k] = string(v.([]byte))
			}
		}
		dst = m
		return nil
	default:
		err := a.storage.Get(dst, query, id)
		if err != nil {
			return err
		}
		return nil
	}
}

func (a *api) insertEntity(table string, rules map[string]string, c *gin.Context) {
	data := make(map[string]interface{})
	if err := c.BindJSON(&data); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	err := a.validator.VarWithValue(data, rules, "map")
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	cols := make([]string, 0, 10)
	vals := make([]interface{}, 0, 10)
	for col, val := range data {
		cols = append(cols, "`"+col+"`")
		vals = append(vals, val)
	}

	placeholder := strings.Repeat("?,", len(cols))
	placeholder = placeholder[:len(placeholder)-1]

	query := fmt.Sprintf(`INSERT INTO %s(%s) VALUES(%s)`, table, strings.Join(cols, ","), placeholder)
	_, err = a.storage.Exec(query, vals...)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (a *api) updateEntity(table string, rules map[string]string, c *gin.Context) {
	if err := a.getEntityByID(c.Param("id"), table, map[string]interface{}{}); err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	updates := make(map[string]interface{})
	if err := c.BindJSON(&updates); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	delete(updates, "id")

	err := a.validator.VarWithValue(updates, rules, "map")
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	cols := make([]string, 0, 10)
	vals := make([]interface{}, 0, 10)
	for col, val := range updates {
		cols = append(cols, "`"+col+"`=?")
		vals = append(vals, val)
	}
	vals = append(vals, c.Param("id"))

	condition := strings.Join(cols, ",")
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", table, condition)

	_, err = a.storage.Exec(query, vals...)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func buildCondition(params queryParams, c *gin.Context) ([]interface{}, []string, error) {
	vals := make([]interface{}, 0, 10)
	cols := make([]string, 0, 10)
	var val interface{}
	var err error

	for paramName, v := range params {
		paramStrVal, exists := c.GetQuery(paramName)
		if paramStrVal == "" && exists {
			return nil, nil, fmt.Errorf("Bad param value %s", paramName)
		}

		if paramStrVal != "" {
			val = paramStrVal
			if v["type"] == "int" {
				val, err = strconv.ParseInt(paramStrVal, 10, 64)
				if err != nil {
					return nil, nil, fmt.Errorf("Bad param value %s", paramName)
				}
			}
			vals = append(vals, val)
			cols = append(cols, v["condition"])
		}
	}

	return vals, cols, nil
}

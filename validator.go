package main

import (
	"math"
	"strconv"
	"strings"

	"github.com/go-playground/validator"
)

const eps = 0.00000000001

func GetValidator() *validator.Validate {
	validate := validator.New()

	validate.RegisterValidation("map", func(fl validator.FieldLevel) bool {
		for _, mapKey := range fl.Top().MapKeys() {
			rule := fl.Top().MapIndex(mapKey).String()
			mapVal := fl.Field().MapIndex(mapKey)
			if !mapVal.IsValid() {
				return !strings.Contains(rule, "required")
			}
			if err := validate.Var(mapVal.Interface(), rule); err != nil {
				return false
			}
		}
		return true
	})

	validate.RegisterValidation("in", func(fl validator.FieldLevel) bool {
		params := strings.Split(fl.Param(), " ")
		verifable := fl.Field()

		if !strings.Contains(verifable.Type().String(), "float") {
			return validate.Var(verifable.Interface(), "oneof="+fl.Param()) == nil
		}

		for _, p := range params {
			v, err := strconv.ParseFloat(p, 64)
			if err != nil {
				panic(err)
			}
			if math.Abs(verifable.Float()-v) < eps {
				return true
			}
		}

		return false
	})

	return validate
}

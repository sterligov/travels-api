package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
)

const maxRowNumberInHub = 20000 // увеличение этого параметра прироста скорости уже не дает
const maxReadingGoroutine = 50

var endJSONEntity = errors.New("End JSON entity")

type DatabaseFiller sqlx.DB

type rawEntity struct {
	cols        string
	placeholder string
	vals        []interface{}
}

func (df *DatabaseFiller) FillDatabaseFromZip(zipFilename string) {
	archive, err := zip.OpenReader(zipFilename)
	if err != nil {
		// log.Fatalln(err)
		panic(err)
	}

	guard := make(chan struct{}, maxReadingGoroutine)
	wg := &sync.WaitGroup{}
	for _, f := range archive.File {
		data, err := f.Open()
		if err != nil {
			//log.Println(err)
			continue
		}
		defer data.Close()

		body, err := ioutil.ReadAll(data)
		if err != nil {
			//log.Println(err)
			continue
		}

		wg.Add(1)
		guard <- struct{}{}
		go func(body []byte) {
			df.parseLineByLineInDatabase(body)
			<-guard
			wg.Done()
		}(body)
	}
	wg.Wait()
	close(guard)
	return
}

func (df *DatabaseFiller) parseLineByLineInDatabase(data []byte) {
	input := bytes.NewReader(data)
	decoder := json.NewDecoder(input)

	_, err := decoder.Token() // {
	if err != nil {
		//log.Println(err)
		return
	}
	entity, err := decoder.Token() // имя сущности
	if err != nil {
		//log.Println(err)
		return
	}
	_, err = decoder.Token() // [
	if err != nil {
		//log.Println(err)
		return
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	hubCh := make(chan *rawEntity)

	go func() {
		defer wg.Done()
		df.hub(entity.(string), hubCh)
	}()

	for {
		re, err := df.parseOneEntity(decoder)
		if err == endJSONEntity {
			continue
		} else if err == io.EOF {
			break
		} else if err != nil {
			//log.Println(err)
		}
		hubCh <- re
	}
	close(hubCh)
	wg.Wait()

	return
}

func (df *DatabaseFiller) parseOneEntity(decoder *json.Decoder) (*rawEntity, error) {
	_, err := decoder.Token() // читаем скобку {
	if err != nil {
		return nil, err
	}

	entity := &rawEntity{
		vals: make([]interface{}, 0, 8),
	}
	var lastColumn string
	var colsBuilder strings.Builder
	var plhBuilder strings.Builder

	// не будем использовать встроенную функцию strings.Join
	// чтобы увеличить скорость считывания, за один проход
	// todo: вообще надо бы проверить насколько это поможет
	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t.(type) {
		case json.Delim:
			entity.placeholder = plhBuilder.String()
			if len(entity.placeholder) > 0 {
				entity.placeholder = entity.placeholder[:len(entity.placeholder)-1]
				entity.cols = colsBuilder.String()
				entity.cols = entity.cols[:len(entity.cols)-1]
			} else {
				return nil, endJSONEntity
			}

			return entity, nil
		default:
			if lastColumn == "" {
				lastColumn = t.(string)
				colsBuilder.WriteString(lastColumn)
				colsBuilder.WriteString(",")
			} else {
				entity.vals = append(entity.vals, t.(interface{}))
				plhBuilder.WriteString("?,")
				lastColumn = ""
			}
		}
	}
}

func (df *DatabaseFiller) hub(entity string, ch <-chan *rawEntity) {
	bucket := make(map[string]*strings.Builder, 0)
	values := make([]interface{}, 0, maxRowNumberInHub)
	counter := 0

	for re := range ch {
		counter++
		if _, ok := bucket[re.cols]; !ok {
			bucket[re.cols] = &strings.Builder{}
		}
		bucket[re.cols].WriteString(`(`)
		bucket[re.cols].WriteString(re.placeholder)
		bucket[re.cols].WriteString(`),`)
		values = append(values, re.vals...)

		if counter == maxRowNumberInHub {
			err := df.insertBucket(entity, values, bucket)
			if err != nil {
				//log.Println(err)
			}

			values = make([]interface{}, 0, maxRowNumberInHub)
			bucket = make(map[string]*strings.Builder, 0)
			counter = 0
		}
	}

	err := df.insertBucket(entity, values, bucket)
	if err != nil {
		//log.Println(err)
	}
}

func (df *DatabaseFiller) insertBucket(entity string, values []interface{}, bucket map[string]*strings.Builder) error {
	for cols, builder := range bucket {
		placeholder := builder.String()
		placeholder = placeholder[:len(placeholder)-1] // удаляем запятую
		query := fmt.Sprintf(`INSERT INTO %s(%s) VALUES%s`,
			entity,
			cols,
			placeholder,
		)
		_, err := df.Exec(query, values...)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}

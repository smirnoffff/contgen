package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"contgen/internal/model"
)

func ParseCSV(r io.Reader) ([]model.Point, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения строк CSV: %w", err)
	}

	if len(header) != 3 {
		return nil, fmt.Errorf("ожидался заголовок year,entity,value")
	}
	for i, value := range header {
		header[i] = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))
	}
	if header[0] != "year" || header[1] != "entity" || header[2] != "value" {
		return nil, fmt.Errorf("ожидался заголовок year,entity,value")
	}
	var points []model.Point
	seen := make(map[string]bool)
	entities := make(map[string]bool)
	for lineNum := 2; ; lineNum++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("строка %d: некорректный CSV: %w", lineNum, err)
		}
		if len(points) >= 10000 {
			return nil, fmt.Errorf("в CSV допускается не более 10000 строк")
		}
		if len(row) != 3 {
			return nil, fmt.Errorf("строка %d: ожидалось 3 колонки, получено %d", lineNum, len(row))
		}
		timeStr := strings.TrimSpace(row[0])
		recordedAt, err := strconv.Atoi(timeStr)
		if err != nil {
			return nil, fmt.Errorf("строка %d: неверный формат года '%s': %w", lineNum, timeStr, err)
		}
		if recordedAt < 1 || recordedAt > 9999 {
			return nil, fmt.Errorf("строка %d: год должен быть от 1 до 9999", lineNum)
		}
		entityName := strings.TrimSpace(row[1])
		if entityName == "" || !utf8.ValidString(entityName) || utf8.RuneCountInString(entityName) > 100 {
			return nil, fmt.Errorf("строка %d: название должно содержать от 1 до 100 символов UTF-8", lineNum)
		}

		valStr := strings.TrimSpace(row[2])
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return nil, fmt.Errorf("строка %d: неверный формат числа '%s': %w", lineNum, valStr, err)
		}
		if math.IsNaN(val) || math.IsInf(val, 0) || val < 0 || val > 9999999999999.99 {
			return nil, fmt.Errorf("строка %d: значение должно быть конечным числом от 0 до 9999999999999.99", lineNum)
		}
		key := strconv.Itoa(recordedAt) + "\x00" + entityName
		if seen[key] {
			return nil, fmt.Errorf("строка %d: повтор сущности %q в %d году", lineNum, entityName, recordedAt)
		}
		seen[key] = true
		entities[entityName] = true
		if len(entities) > 100 {
			return nil, fmt.Errorf("в датасете допускается не более 100 сущностей")
		}

		points = append(points, model.Point{
			EntityName: entityName,
			Value:      val,
			RecordedAt: recordedAt,
		})
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("CSV не содержит данных")
	}
	return points, nil

}

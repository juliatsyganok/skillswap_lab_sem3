package httpapi

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

var ErrInvalidQuery = errors.New("invalid query")

type OffersQuery struct {
	Skill  string
	Limit  int
	Offset int
}

func ParseOffersQuery(values url.Values) (OffersQuery, error) {
	result := OffersQuery{
		Limit:  20,
		Offset: 0,
	}

	allowed := map[string]bool{
		"skill":  true,
		"limit":  true,
		"offset": true,
	}

	for key, items := range values {
		if !allowed[key] || len(items) != 1 {
			return OffersQuery{}, ErrInvalidQuery
		}
	}

	if items, ok := values["skill"]; ok {
		result.Skill = strings.ToLower(strings.TrimSpace(items[0]))

		if result.Skill == "" {
			return OffersQuery{}, ErrInvalidQuery
		}
	}

	if items, ok := values["limit"]; ok {
		n, err := strconv.Atoi(items[0])

		if err != nil || n < 1 || n > 50 {
			return OffersQuery{}, ErrInvalidQuery
		}

		result.Limit = n
	}

	if items, ok := values["offset"]; ok {
		n, err := strconv.Atoi(items[0])

		if err != nil || n < 0 {
			return OffersQuery{}, ErrInvalidQuery
		}

		result.Offset = n
	}

	return result, nil
}

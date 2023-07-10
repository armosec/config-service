package handlers

import (
	"config-service/db"
	"config-service/utils"
	"config-service/utils/consts"
	"fmt"
	"strings"

	"github.com/armosec/armoapi-go/armotypes"
)

const (
	existsFilter   string = "exists"
	missingFilter  string = "missing"
	matchFilter    string = "match"
	greaterFilter  string = "greater"
	lowerFilter    string = "lower"
	ascendingSort  string = "asc"
	descendingSort string = "desc"
)

const maxV2PageSize = 1000

func v2List2FindOptions(request armotypes.V2ListRequest) (*db.FindOptions, error) {
	findOptions := db.NewFindOptions()
	//pages
	var perPage int
	if request.PageSize != nil && *request.PageSize <= maxV2PageSize {
		perPage = *request.PageSize
	}
	var page int
	if request.PageNum != nil {
		page = *request.PageNum
	}
	findOptions.SetPagination(int64(page), int64(perPage))

	//sort
	if request.OrderBy == "" {
		//default sort by update time
		request.OrderBy = fmt.Sprintf("%s:%s", consts.UpdatedTimeField, descendingSort)
	}
	sortFields := strings.Split(request.OrderBy, ",")
	for _, sortField := range sortFields {
		sortNameAndType := strings.Split(sortField, ":")
		if len(sortNameAndType) != 2 {
			return nil, fmt.Errorf("invalid sort field %s", sortField)
		}
		switch sortNameAndType[1] {
		case ascendingSort:
			findOptions.Sort().AddAscending(sortNameAndType[0])
		case descendingSort:
			findOptions.Sort().AddDescending(sortNameAndType[0])
		default:
			return nil, fmt.Errorf("invalid sort type %s", sortNameAndType[1])
		}
	}

	//projection
	findOptions.Projection().Include(request.FieldsList...)

	//filters
	if len(request.InnerFilters) > 0 {
		filters := make([]*db.FilterBuilder, len(request.InnerFilters))
		for i := range request.InnerFilters {
			if filter, err := buildInnerFilter(request.InnerFilters[i]); err != nil {
				return nil, err
			} else if filter != nil {
				filters[i] = filter
			}
		}
		if len(filters) > 1 {
			findOptions.Filter().AddOr(filters...)
		} else {
			findOptions.Filter().WithFilter(filters[0])
		}
	}

	return findOptions, nil
}

func buildInnerFilter(innerFilter map[string]string) (*db.FilterBuilder, error) {
	filterBuilder := db.NewFilterBuilder()
	for key, value := range innerFilter {
		valueAndOperation := strings.Split(value, "|")
		values := strings.Split(valueAndOperation[0], ",")
		operation := matchFilter
		if len(valueAndOperation) == 2 {
			operation = valueAndOperation[1]
		}
		switch operation {
		case existsFilter:
			filterBuilder.AddExists(key, true)

		case missingFilter:
			filterBuilder.AddExists(key, false)

		case matchFilter:
			filters := make([]*db.FilterBuilder, len(values))
			isID := key == consts.GUIDField
			for i, value := range values {
				if isID {
					filters[i] = db.NewFilterBuilder().WithID(value)
				} else {
					filters[i] = db.NewFilterBuilder().WithValue(key, utils.String2Interface(value))
				}
			}
			if len(filters) > 1 {
				filterBuilder.AddOr(filters...)
			} else {
				filterBuilder.WithFilter(filters[0])
			}

		case greaterFilter:
			filters := make([]*db.FilterBuilder, len(values))
			for i, value := range values {
				filters[i] = db.NewFilterBuilder().WithGreaterThanEqual(key, utils.String2Interface(value))
			}
			if len(filters) > 1 {
				filterBuilder.AddOr(filters...)
			} else {
				filterBuilder.WithFilter(filters[0])
			}

		case lowerFilter:
			filters := make([]*db.FilterBuilder, len(values))
			for i, value := range values {
				filters[i] = db.NewFilterBuilder().WithLowerThanEqual(key, utils.String2Interface(value))
			}
			if len(filters) > 1 {
				filterBuilder.AddOr(filters...)
			} else {
				filterBuilder.WithFilter(filters[0])
			}

		default:
			return nil, fmt.Errorf("unsupported operation %s", operation)
		}

	}
	if filterBuilder.Len() == 0 {
		return nil, nil
	}
	return filterBuilder, nil
}

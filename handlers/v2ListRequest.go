package handlers

import (
	"config-service/db"
	"config-service/utils"
	"config-service/utils/consts"
	"fmt"
	"strings"

	"github.com/armosec/armoapi-go/armotypes"
)

// TODO:
// support 'Until' (TBD according to what time - creation time or update time)
// Add missing operator "like" for wildcards (need to transform to regex)
// support array element match - needs some schema information of what are searchable arrays)
// support time range for fields of time.time type (vs. RFC3339 string) - need to add schema information
const (
	existsOperator   string = "exists"
	equalOperator    string = "equal"
	missingOperator  string = "missing"
	matchOperator    string = "match"
	greaterOperator  string = "greater"
	lowerOperator    string = "lower"
	regexOperator    string = "regex"
	rangeOperator    string = "range"
	ignoreCaseOption string = "ignorecase"

	ascendingSort  string = "asc"
	descendingSort string = "desc"

	valueSeparator    = ","
	operatorSeparator = "|"
	subQuerySeparator = "&"
	sortTypeSeparator = ":"
	escapeChar        = "\\"
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
	sortFields := strings.Split(request.OrderBy, valueSeparator)
	for _, sortField := range sortFields {
		sortNameAndType := strings.Split(sortField, sortTypeSeparator)
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
		// Split the value into parts by comma
		parts := utils.SplitIgnoreEscaped(value, valueSeparator, escapeChar)
		//parts := strings.Split(value, valueSeparator)
		// Prepare a slice to hold all filters for this key
		filters := make([]*db.FilterBuilder, 0, len(parts))
		for _, part := range parts {
			// Split each part into value and operator by pipe
			valueAndOperation := strings.Split(part, operatorSeparator)
			value := valueAndOperation[0]
			value = strings.ReplaceAll(value, escapeChar, "")
			operator := matchOperator
			operatorOption := ""
			if len(valueAndOperation) == 2 {
				operatorAndOption := strings.Split(valueAndOperation[1], subQuerySeparator)
				operator = operatorAndOption[0]
				if len(operatorAndOption) == 2 {
					operatorOption = operatorAndOption[1]
				}
			}
			switch operator {
			case existsOperator:
				filters = append(filters, db.NewFilterBuilder().AddExists(key, true))
			case missingOperator:
				filters = append(filters, db.NewFilterBuilder().AddExists(key, false))
			case matchOperator, equalOperator:
				if key == consts.GUIDField {
					filters = append(filters, db.NewFilterBuilder().WithID(value))
				} else {
					filters = append(filters, db.NewFilterBuilder().WithValue(key, utils.String2Interface(value)))
				}
			case greaterOperator:
				filters = append(filters, db.NewFilterBuilder().WithGreaterThanEqual(key, utils.String2Interface(value)))
			case lowerOperator:
				filters = append(filters, db.NewFilterBuilder().WithLowerThanEqual(key, utils.String2Interface(value)))
			case regexOperator:
				ignoreCase := operatorOption == ignoreCaseOption
				filters = append(filters, db.NewFilterBuilder().WithRegex(key, value, ignoreCase))
			case rangeOperator:
				rangeValues := strings.Split(value, subQuerySeparator)
				if len(rangeValues) != 2 {
					return nil, fmt.Errorf("value missing range separator %s", value)
				}
				if rangeValues[0] == "" || rangeValues[1] == "" {
					return nil, fmt.Errorf("invalid range value %s", value)
				}
				val1 := utils.String2Interface(rangeValues[0])
				val2 := utils.String2Interface(rangeValues[1])
				if !utils.SameType(val1, val2) {
					return nil, fmt.Errorf("invalid range must use same value types %s", value)
				}
				filters = append(filters, db.NewFilterBuilder().WithRange(key, val1, val2))
			default:
				return nil, fmt.Errorf("unsupported operator %s", operator)
			}
		}
		// Add all filters for this key to the main filter builder
		if len(filters) > 1 {
			filterBuilder.AddOr(filters...)
		} else if len(filters) == 1 {
			filterBuilder.WithFilter(filters[0])
		}
	}
	if filterBuilder.Len() == 0 {
		return nil, nil
	}
	return filterBuilder, nil
}

func buildInnerFilter1(innerFilter map[string]string) (*db.FilterBuilder, error) {
	filterBuilder := db.NewFilterBuilder()
	for key, value := range innerFilter {
		valueAndOperation := strings.Split(value, operatorSeparator)
		values := strings.Split(valueAndOperation[0], valueSeparator)
		operator := matchOperator
		if len(valueAndOperation) == 2 {
			operator = valueAndOperation[1]
		}
		switch operator {
		case existsOperator:
			filterBuilder.AddExists(key, true)

		case missingOperator:
			filterBuilder.AddExists(key, false)

		case matchOperator:
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

		case greaterOperator:
			filters := make([]*db.FilterBuilder, len(values))
			for i, value := range values {
				filters[i] = db.NewFilterBuilder().WithGreaterThanEqual(key, utils.String2Interface(value))
			}
			if len(filters) > 1 {
				filterBuilder.AddOr(filters...)
			} else {
				filterBuilder.WithFilter(filters[0])
			}

		case lowerOperator:
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
			return nil, fmt.Errorf("unsupported operator %s", operator)
		}

	}
	if filterBuilder.Len() == 0 {
		return nil, nil
	}
	return filterBuilder, nil
}

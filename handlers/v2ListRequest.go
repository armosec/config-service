package handlers

import (
	"config-service/db"
	"config-service/utils"
	"config-service/utils/consts"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/armosec/armoapi-go/armotypes"
)

const maxV2PageSize = 150

func v2List2FindOptions(ctx *gin.Context, request armotypes.V2ListRequest) (*db.FindOptions, error) {
	request.ValidatePageProperties(maxV2PageSize)
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
	tsField := db.GetSchemaFromContext(ctx).GetTimestampFieldName()
	request.ValidateOrderBy(fmt.Sprintf("%s:%s", tsField, armotypes.V2ListDescendingSort))
	sortFields := strings.Split(request.OrderBy, armotypes.V2ListValueSeparator)
	for _, sortField := range sortFields {
		sortNameAndType := strings.Split(sortField, armotypes.V2ListSortTypeSeparator)
		if len(sortNameAndType) != 2 {
			return nil, fmt.Errorf("invalid sort field %s", sortField)
		}
		switch sortNameAndType[1] {
		case armotypes.V2ListAscendingSort:
			findOptions.Sort().AddAscending(sortNameAndType[0])
		case armotypes.V2ListDescendingSort:
			findOptions.Sort().AddDescending(sortNameAndType[0])
		default:
			return nil, fmt.Errorf("invalid sort type %s", sortNameAndType[1])
		}
	}
	//projection
	findOptions.Projection().Include(request.FieldsList...)
	if len(request.FieldsList) == 0 {
		findOptions.Projection().Exclude(db.GetSchemaFromContext(ctx).GetMustExcludeFields()...)
	}
	//filters
	if request.Until != nil {
		timeVal := request.Until.Format(time.RFC3339)
		value, err := getTypedValue(ctx, tsField, timeVal)
		if err != nil {
			return nil, err
		}
		findOptions.Filter().WithLowerThanEqual(tsField, value)
	}
	if request.Since != nil {
		timeVal := request.Since.Format(time.RFC3339)
		value, err := getTypedValue(ctx, tsField, timeVal)
		if err != nil {
			return nil, err
		}
		findOptions.Filter().WithGreaterThanEqual(tsField, value)
	}
	if len(request.InnerFilters) > 0 {
		filters := []*db.FilterBuilder{}
		for i := range request.InnerFilters {
			if filter, err := buildInnerFilter(ctx, request.InnerFilters[i], ""); err != nil {
				return nil, err
			} else if filter != nil {
				filters = append(filters, filter)
			}
		}
		if len(filters) > 1 {
			findOptions.Filter().AddOr(filters...)
		} else if len(filters) == 1 {
			findOptions.Filter().WithFilter(filters[0])
		}
	}
	return findOptions, nil
}

func uniqueValuesRequest2FindOptions(ctx *gin.Context, request armotypes.UniqueValuesRequestV2) (*db.FindOptions, error) {
	request.ValidatePageProperties(maxV2PageSize)
	if len(request.Fields) == 0 {
		return nil, fmt.Errorf("fields are required")
	}
	findOptions := db.NewFindOptions()
	//pages
	var page int
	if request.PageNum != nil {
		page = *request.PageNum
	}
	findOptions.SetPagination(int64(page), int64(request.PageSize))
	for field := range request.Fields {
		findOptions.WithGroup(field)
	}
	findOptions.Limit(int64(request.PageSize))
	//filters
	tsField := db.GetSchemaFromContext(ctx).GetTimestampFieldName()
	if request.Until != nil {
		findOptions.Filter().WithLowerThanEqual(tsField, *request.Until)
	}
	if request.Since != nil {
		findOptions.Filter().WithGreaterThanEqual(tsField, *request.Since)
	}
	if len(request.InnerFilters) > 0 {
		filters := []*db.FilterBuilder{}
		for i := range request.InnerFilters {
			if filter, err := buildInnerFilter(ctx, request.InnerFilters[i], ""); err != nil {
				return nil, err
			} else if filter != nil {
				filters = append(filters, filter)
			}
		}
		if len(filters) > 1 {
			findOptions.Filter().AddOr(filters...)
		} else if len(filters) == 1 {
			findOptions.Filter().WithFilter(filters[0])
		}
	}
	return findOptions, nil
}

// buildInnerFilter builds a filter from a map of key value pairs
// if it calls itself recursively (e.g. for element match operator) the rootField must be the array field path
func buildInnerFilter(ctx *gin.Context, innerFilter map[string]string, rootField string) (*db.FilterBuilder, error) {
	filterBuilder := db.NewFilterBuilder()
	schemaInfo := db.GetSchemaFromContext(ctx)
	var elemMatches map[string]map[string]string
	for key, value := range innerFilter {
		//ignore empty values
		if value == "" {
			continue
		}
		//check if key has element match operator
		keyNOperator := strings.Split(key, armotypes.V2ListOperatorSeparator)
		if len(keyNOperator) > 1 && keyNOperator[1] == armotypes.V2ListElementMatchOperator {
			key = keyNOperator[0]
			keyWithRoot := key
			if rootField != "" {
				keyWithRoot = fmt.Sprintf("%s.%s", rootField, key)
			}
			isArray, arrayPath, fieldPath := schemaInfo.GetArrayDetails(keyWithRoot)
			if !isArray {
				return nil, fmt.Errorf("element match operator is only supported for array fields")
			}
			if elemMatches == nil {
				elemMatches = map[string]map[string]string{}
			}
			if _, ok := elemMatches[arrayPath]; !ok {
				elemMatches[arrayPath] = map[string]string{}
			}
			elemMatches[arrayPath][fieldPath] = value
			continue
		}
		// Split the value into parts by comma
		parts := utils.SplitIgnoreEscaped(value, armotypes.V2ListValueSeparator, armotypes.V2ListEscapeChar)
		//parts := strings.Split(value, valueSeparator)
		// Prepare a slice to hold all filters for this key
		filters := make([]*db.FilterBuilder, 0, len(parts))
		for _, part := range parts {
			valueAndOperation := strings.Split(part, armotypes.V2ListOperatorSeparator)
			value := valueAndOperation[0]
			value = strings.ReplaceAll(value, armotypes.V2ListEscapeChar, "")
			operator := armotypes.V2ListMatchOperator
			operatorOption := ""
			if len(valueAndOperation) == 2 {
				operatorAndOption := strings.Split(valueAndOperation[1], armotypes.V2ListSubQuerySeparator)
				operator = operatorAndOption[0]
				if len(operatorAndOption) == 2 {
					operatorOption = operatorAndOption[1]
				}
			}
			switch operator {
			case armotypes.V2ListExistsOperator:
				filters = append(filters, db.NewFilterBuilder().AddExists(key, true))
			case armotypes.V2ListMissingOperator:
				filters = append(filters, db.NewFilterBuilder().AddExists(key, false))
			case armotypes.V2ListMatchOperator, armotypes.V2ListEqualOperator:
				if key == consts.GUIDField {
					filters = append(filters, db.NewFilterBuilder().WithID(value))
				} else {
					//append root field if exists
					keyWithRoot := key
					if rootField != "" {
						keyWithRoot = fmt.Sprintf("%s.%s", rootField, key)
					}
					value, err := getTypedValue(ctx, keyWithRoot, value)
					if err != nil {
						return nil, err
					}
					filters = append(filters, db.NewFilterBuilder().WithValue(key, value))
				}
			case armotypes.V2ListGreaterOperator:
				value, err := getTypedValue(ctx, key, value)
				if err != nil {
					return nil, err
				}
				filters = append(filters, db.NewFilterBuilder().WithGreaterThanEqual(key, value))
			case armotypes.V2ListLowerOperator:
				value, err := getTypedValue(ctx, key, value)
				if err != nil {
					return nil, err
				}
				filters = append(filters, db.NewFilterBuilder().WithLowerThanEqual(key, value))
			case armotypes.V2ListLikeOperator:
				value = regexp.QuoteMeta(value)
				fallthrough
			case armotypes.V2ListRegexOperator:
				ignoreCase := operatorOption == armotypes.V2ListIgnoreCaseOption
				filters = append(filters, db.NewFilterBuilder().WithRegex(key, value, ignoreCase))
			case armotypes.V2ListRangeOperator:
				rangeValues := strings.Split(value, armotypes.V2ListSubQuerySeparator)
				if len(rangeValues) != 2 {
					return nil, fmt.Errorf("value missing range separator %s", value)
				}
				if rangeValues[0] == "" || rangeValues[1] == "" {
					return nil, fmt.Errorf("invalid range value %s", value)
				}
				val1, err := getTypedValue(ctx, key, rangeValues[0])
				if err != nil {
					return nil, err
				}
				val2, err := getTypedValue(ctx, key, rangeValues[1])
				if err != nil {
					return nil, err
				}
				if !utils.SameType(val1, val2) {
					return nil, fmt.Errorf("invalid range must use same value types found %T %T", val1, val2)
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
	//add element match filters
	for array, elemInnerFilters := range elemMatches {
		elemFilter, err := buildInnerFilter(ctx, elemInnerFilters, array)
		if err != nil {
			return nil, fmt.Errorf("invalid element match filters %v", err)
		}
		filterBuilder.WithFilter(elemFilter.WarpElementMatch().WarpWithField(array))
	}
	if filterBuilder.Len() == 0 {
		return nil, nil
	}
	return filterBuilder, nil
}

func getTypedValue(ctx *gin.Context, field, value string) (interface{}, error) {
	schemaInfo := db.GetSchemaFromContext(ctx)
	if schemaInfo.IsString(field) {
		return value, nil
	}
	if schemaInfo.IsDate(field) {
		date, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field %s with value %s into Time type", field, value)
		}
		return date, nil
	}
	return utils.String2Interface(value), nil
}

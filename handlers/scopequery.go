package handlers

import (
	"config-service/db"
	"config-service/utils/consts"
	"config-service/utils/log"
	"context"
	"net/url"
	"strconv"
	"strings"

	"k8s.io/utils/strings/slices"
)

type QueryParamsConfig struct {
	Params2Query   map[string]QueryConfig
	DefaultContext string
}

type QueryConfig struct {
	FieldName   string
	PathInArray string
	IsArray     bool
	ParseValue  bool //should parse string to int/bool
}

func DefaultQueryConfig() *QueryParamsConfig {
	return &QueryParamsConfig{
		DefaultContext: "attributes",
		Params2Query: map[string]QueryConfig{
			"attributes": {
				FieldName:   "attributes",
				PathInArray: "",
				IsArray:     false,
			},
		},
	}
}

// flat query config - for query params that are not nested
func FlatQueryConfig() *QueryParamsConfig {
	return &QueryParamsConfig{
		Params2Query: map[string]QueryConfig{
			"": {
				FieldName:   "",
				PathInArray: "",
				IsArray:     false,
			},
		},
		DefaultContext: "",
	}
}

// QueryParams2Filter build a query filter from query params
func QueryParams2Filter(c context.Context, qParams url.Values, conf *QueryParamsConfig) *db.FilterBuilder {
	if conf == nil {
		return nil
	}
	//keep filter builder per field name
	filterBuilders := map[string]*db.FilterBuilder{}
	getFilterBuilder := func(paramName string) *db.FilterBuilder {
		if filterBuilder, ok := filterBuilders[paramName]; ok {
			return filterBuilder
		}
		filterBuilder := db.NewFilterBuilder()
		filterBuilders[paramName] = filterBuilder
		return filterBuilder
	}

	for paramKey, vals := range qParams {
		if !isSearchParam(paramKey) {
			continue
		}
		keys := strings.Split(paramKey, ".")
		//clean whitespaces
		values := slices.Filter([]string{}, vals, func(s string) bool { return s != "" })
		if len(values) == 0 {
			continue
		}
		if len(keys) < 2 {
			keys = []string{conf.DefaultContext, keys[0]}
		} else if len(keys) > 2 {
			keys = []string{keys[0], strings.Join(keys[1:], ".")}
		}
		//escape in case of bad formatted query params
		for i := range values {
			if v, err := url.QueryUnescape(values[i]); err != nil {
				log.LogNTraceError("failed to unescape query param", err, c)
			} else {
				values[i] = v
			}
		}
		//calculate field name
		var field, key = keys[0], keys[1]
		queryConfig, ok := conf.Params2Query[field]
		if !ok {
			queryConfig = QueryConfig{FieldName: field, ParseValue: true}
		}
		if queryConfig.IsArray {
			if queryConfig.PathInArray != "" {
				key = queryConfig.PathInArray + "." + key
			}
		} else if queryConfig.FieldName != "" {
			key = queryConfig.FieldName + "." + key
		}
		//get the field filter builder
		filterBuilder := getFilterBuilder(queryConfig.FieldName)
		//case of single value
		if len(values) == 1 {
			addValue(filterBuilder, queryConfig, key, values[0])
		} else { //case of multiple values
			fb := db.NewFilterBuilder()
			for _, v := range values {
				addValue(fb, queryConfig, key, v)
			}
			filterBuilder.WithFilter(fb.WarpOr().Get())
		}
	}
	//aggregate all filters
	allQueriesFilter := db.NewFilterBuilder()
	for key, filterBuilder := range filterBuilders {
		filterBuilder.WrapDupKeysWithOr()
		queryConfig, ok := conf.Params2Query[key]
		if ok && queryConfig.IsArray {
			filterBuilder.WarpElementMatch().WarpWithField(queryConfig.FieldName)
		}
		allQueriesFilter.WithFilter(filterBuilder.Get())
	}
	if len(allQueriesFilter.Get()) == 0 {
		return nil
	}
	return allQueriesFilter
}

func isSearchParam(param string) bool {
	switch param {
	case consts.CustomerGUID, consts.LimitParam, consts.SkipParam,
		consts.FromDateParam, consts.ToDateParam, consts.ProjectionParam:
		return false
	default:
		return true
	}
}

func addValue(filterBuilder *db.FilterBuilder, queryConfig QueryConfig, key, value string) {
	if !queryConfig.ParseValue {
		filterBuilder.WithValue(key, value)
		return
	}
	if b, err := strconv.ParseBool(value); err == nil {
		filterBuilder.WithEqual(key, b)
		return
	}
	if i, err := strconv.Atoi(value); err == nil {
		filterBuilder.WithEqual(key, i)
		return
	}
	filterBuilder.WithValue(key, value)
}

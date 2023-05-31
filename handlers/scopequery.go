package handlers

import (
	"config-service/db"
	"config-service/utils/log"
	"context"
	"net/url"
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
		QueryConfig, ok := conf.Params2Query[field]
		if !ok {
			continue
		} else if QueryConfig.IsArray {
			if QueryConfig.PathInArray != "" {
				key = QueryConfig.PathInArray + "." + key
			}
		} else if QueryConfig.FieldName != "" {
			key = QueryConfig.FieldName + "." + key
		}
		//get the field filter builder
		filterBuilder := getFilterBuilder(QueryConfig.FieldName)
		//case of single value
		if len(values) == 1 {
			filterBuilder.WithValue(key, values[0])
		} else { //case of multiple values
			fb := db.NewFilterBuilder()
			for _, v := range values {
				fb.WithValue(key, v)
			}
			filterBuilder.WithFilter(fb.WarpOr().Get())
		}
	}
	//aggregate all filters
	allQueriesFilter := db.NewFilterBuilder()
	for key, filterBuilder := range filterBuilders {
		QueryConfig := conf.Params2Query[key]
		filterBuilder.WrapDupKeysWithOr()
		if QueryConfig.IsArray {
			filterBuilder.WarpElementMatch().WarpWithField(QueryConfig.FieldName)
		}
		allQueriesFilter.WithFilter(filterBuilder.Get())
	}
	if len(allQueriesFilter.Get()) == 0 {
		return nil
	}
	return allQueriesFilter
}

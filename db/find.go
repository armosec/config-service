package db

type FindOptions struct {
	filter     *FilterBuilder
	projection *ProjectionBuilder
	sort       *SortBuilder
	group      []string
	limit      int64
	skip       int64
}

func NewFindOptions() *FindOptions {
	return &FindOptions{
		filter:     NewFilterBuilder(),
		projection: NewProjectionBuilder(),
		sort:       NewSortBuilder(),
	}
}

func (f *FindOptions) WithGroup(group ...string) *FindOptions {
	f.group = append(f.group, group...)
	return f
}

func (f *FindOptions) GetGroup() []string {
	return f.group
}

func (f *FindOptions) WithFilter(filter *FilterBuilder) *FindOptions {
	f.filter = filter
	return f
}

func (f *FindOptions) Filter() *FilterBuilder {
	return f.filter
}

func (f *FindOptions) WithProjection(projection *ProjectionBuilder) *FindOptions {
	f.projection = projection
	return f
}

func (f *FindOptions) Projection() *ProjectionBuilder {
	return f.projection
}

func (f *FindOptions) WithSort(sort *SortBuilder) *FindOptions {
	f.sort = sort
	return f
}

func (f *FindOptions) Sort() *SortBuilder {
	return f.sort
}

func (f *FindOptions) Limit(limit int64) *FindOptions {
	f.limit = limit
	return f
}

func (f *FindOptions) Skip(skip int64) *FindOptions {
	f.skip = skip
	return f
}

func (f *FindOptions) SetPagination(page, perPage int64) *FindOptions {
	if page < 1 {
		page = 1
	}
	f.skip = (page - 1) * perPage
	f.limit = perPage
	return f
}

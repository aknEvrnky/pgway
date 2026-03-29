package domain

const DefaultMaxPageSize = 100

type ListParams struct {
	PageSize int
	Cursor   string
}

type ListResult[T any] struct {
	Items      []*T
	NextCursor string
	TotalCount int
}

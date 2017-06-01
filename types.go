package qb

import (
	"database/sql"
	"database/sql/driver"
	"time"
)

///
/// Source
///

// Query ...
type Query interface {
	SQL() (string, []interface{})
}

type query struct {
	sql    string
	values []interface{}
}

func newQuery(s string, v []interface{}) Query {
	return &query{sql: s, values: v}
}

// SQL ...
func (q *query) SQL() (string, []interface{}) {
	return q.sql, q.values
}

// Alias ...
type Alias interface {
	Get(Source) string
}

// Source ...
type Source interface {
	QueryString(Alias, *ValueList) string
	AliasString() string
}

// Table ...
type Table struct {
	Name string
}

// QueryString ...
func (t *Table) QueryString(ag Alias, _ *ValueList) string {
	alias := ag.Get(t)
	if len(alias) > 0 {
		alias = ` ` + alias
	}
	return t.Name + alias
}

// AliasString ...
func (t *Table) AliasString() string {
	return `t`
}

// Select ...
func (t *Table) Select(f ...DataField) SelectBuilder {
	return NewSelectBuilder(f, t)
}

// Delete ...
func (t *Table) Delete(c1 Condition, c ...Condition) Query {
	return newQuery(DeleteSQL(t, append(c, c1)))
}

// Update ...
func (t *Table) Update() UpdateBuilder {
	return UpdateBuilder{t, nil, nil}
}

// SubQuery ...
type SubQuery struct {
	sql    string
	values []interface{}
	fields []Field
}

// QueryString ...
func (t *SubQuery) QueryString(ag Alias, vl *ValueList) string {
	alias := ag.Get(t)
	if len(alias) > 0 {
		alias = ` ` + alias
	}
	return `(` + t.sql + `)` + alias
}

// AliasString ...
func (t *SubQuery) AliasString() string {
	return `sq`
}

// Select ...
func (t *SubQuery) Select(f ...DataField) SelectBuilder {
	return NewSelectBuilder(f, t)
}

///
/// Field
///

// Field represents a field in a query
type Field interface {
	QueryString(Alias, *ValueList) string
	Source() Source
	DataType() string
}

// Cursor ...
type Cursor struct {
	fields             []DataField
	rows               *sql.Rows
	DisableExitOnError bool
	err                error
}

// NewCursor returns a new Cursor
func NewCursor(f []DataField, r *sql.Rows) *Cursor {
	c := &Cursor{fields: f, rows: r}
	return c
}

// Next loads the next row into the fields
func (c *Cursor) Next() bool {
	if !c.rows.Next() {
		c.Close()
		return false
	}
	err := ScanToFields(c.fields, c.rows)
	if err != nil {
		c.err = err
		if !c.DisableExitOnError {
			c.Close()
		}
		return false
	}

	return true
}

// Close the rows object, automatically called by Next when all rows have been read
func (c *Cursor) Close() {
	_ = c.rows.Close()
}

// Error returns the last error
func (c Cursor) Error() error {
	return c.err
}

// TableField represents a real field in a table
type TableField struct {
	Parent     Source
	Name       string
	Type       string
	ReadOnly   bool
	HasDefault bool
	Primary    bool
}

// QueryString ...
func (f TableField) QueryString(ag Alias, _ *ValueList) string {
	alias := ag.Get(f.Parent)
	if alias != `` {
		alias += `.`
	}
	return alias + f.Name
}

// Source ...
func (f TableField) Source() Source {
	return f.Parent
}

// DataType ...
func (f TableField) DataType() string {
	return f.Type
}

// New creates a new instance of the field with a different Parent
func (f TableField) New(src Source) *TableField {
	f.Parent = src
	return &f
}

// ValueField contains values supplied by the program
type ValueField struct {
	Value interface{}
	Type  string
}

func getValue(v interface{}) interface{} {
	if val, ok := v.(driver.Valuer); ok {
		new, err := val.Value()
		if err == nil {
			v = new
		}
	}
	return v
}

func getType(v interface{}) string {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return `int`
	case float32, float64:
		return `float`
	case string:
		return `string`
	case bool:
		return `bool`
	case time.Time:
		return `time`
	default:
		return `unknown`
	}
}

// Value ...
func Value(v interface{}) ValueField {
	v = getValue(v)
	return ValueField{v, getType(v)}
}

// QueryString ...
func (f ValueField) QueryString(_ Alias, vl *ValueList) string {
	vl.Append(f.Value)
	return VALUE
}

// Source ...
func (f ValueField) Source() Source {
	return nil
}

// DataType ...
func (f ValueField) DataType() string {
	return f.Type
}

///
/// Query types
///

// Condition is used in the Where function
type Condition func(Alias, *ValueList) string

// Join are used for joins on tables
type Join struct {
	Type       string
	Fields     [2]Field
	New        Source
	Conditions []Condition
}

// FieldOrder specifies the order in which fields should be sorted
type FieldOrder struct {
	Field Field
	Order string
}

// Asc is used to sort in ascending order
func Asc(f Field) FieldOrder {
	return FieldOrder{Field: f, Order: `ASC`}
}

// Desc is used to sort in descending order
func Desc(f Field) FieldOrder {
	return FieldOrder{Field: f, Order: `DESC`}
}

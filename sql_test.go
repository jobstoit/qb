package qb

import (
	"testing"

	"git.ultraware.nl/NiseVoid/qb/tests/testutil"
)

func newCheckOutput(t *testing.T, b *sqlBuilder) func(bool, string) {
	return func(newline bool, expected string) {
		out := b.w.String()
		b.w = sqlWriter{}

		if newline {
			expected += "\n"
		}

		testutil.Compare(t, expected, out)
	}
}

func info(t *testing.T, msg string) {
	t.Log(testutil.Notice(msg))
}

func testBuilder(t *testing.T, alias bool) (*sqlBuilder, func(bool, string)) {
	b := &sqlBuilder{sqlWriter{}, nil, NoAlias(), nil}
	if alias {
		b.alias = AliasGenerator()
	}
	return b, newCheckOutput(t, b)
}

// Tables

var testTable = &Table{Name: `tmp`}
var testFieldA = &TableField{Name: `colA`, Parent: testTable}
var testFieldB = &TableField{Name: `colB`, Parent: testTable}

var testTable2 = &Table{Name: `tmp2`}
var testFieldA2 = &TableField{Name: `colA2`, Parent: testTable2}

func NewIntField(f Field) DataField {
	i := 0
	return NewDataField(f, &i)
}

func TestFrom(t *testing.T) {
	info(t, `-- Testing without alias`)
	b, check := testBuilder(t, false)

	b.From(testTable)
	check(true, `FROM tmp`)

	info(t, `-- Testing with alias`)
	b, check = testBuilder(t, true)

	b.From(testTable)
	check(true, `FROM tmp t1`)
}

func TestDelete(t *testing.T) {
	b, check := testBuilder(t, false)

	b.Delete(testTable)
	check(true, `DELETE FROM tmp`)
}

func TestUpdate(t *testing.T) {
	b, check := testBuilder(t, false)

	b.Update(testTable)
	check(true, `UPDATE tmp`)
}

func TestInsert(t *testing.T) {
	b, check := testBuilder(t, false)

	b.Insert(testTable, nil)
	check(true, `INSERT INTO tmp ()`)

	b.Insert(testTable, []DataField{NewIntField(testFieldA), NewIntField(testFieldB)})
	check(true, `INSERT INTO tmp (colA, colB)`)
}

func TestJoin(t *testing.T) {
	b, check := testBuilder(t, true)

	b.From(testTable)
	b.w = sqlWriter{}

	b.Join(join{JoinInner, testTable2, []Condition{eq(testFieldA, testFieldA2)}})
	check(true,
		"\t"+`INNER JOIN tmp2 t2 ON (t1.colA = t2.colA2)`,
	)

	b.Join(join{JoinLeft, testTable2, []Condition{eq(testFieldA, testFieldA2), testCondition, testCondition2}})
	check(true,
		"\t"+`LEFT JOIN tmp2 t2 ON (t1.colA = t2.colA2 AND a AND b)`,
	)

	b.Join(
		join{JoinInner, testTable2, []Condition{eq(testFieldA, testFieldA2)}},
		join{JoinLeft, testTable2, []Condition{eq(testFieldA, testFieldA2), testCondition, testCondition2}},
	)
	check(true,
		"\t"+`INNER JOIN tmp2 t2 ON (t1.colA = t2.colA2)`+"\n\t"+
			`LEFT JOIN tmp2 t2 ON (t1.colA = t2.colA2 AND a AND b)`,
	)
}

// Fields

func TestSelect(t *testing.T) {
	f1 := NewIntField(testFieldA)
	f2 := NewIntField(testFieldB)

	info(t, `-- Testing without alias`)
	b, check := testBuilder(t, false)

	b.Select(false, f1, f2)
	check(true, `SELECT colA, colB`)

	b.Select(true, f1, f2)
	check(true, `SELECT colA f0, colB f1`)

	info(t, `-- Testing with alias`)
	b, check = testBuilder(t, true)

	b.Select(false, f1, f2)
	check(true, `SELECT t1.colA, t1.colB`)

	b.Select(true, f1, f2)
	check(true, `SELECT t1.colA f0, t1.colB f1`)
}

func TestSet(t *testing.T) {
	b, check := testBuilder(t, false)

	b.Set([]set{})
	check(false, ``)

	b.Set([]set{{testFieldA, Value(1)}})
	check(true, `SET colA = ?`)

	b.Set([]set{{testFieldA, Value(1)}, {testFieldB, Value(3)}})
	check(true, "SET\n\tcolA = ?,\n\tcolB = ?")
}

// Conditions

var testCondition = func(_ Driver, _ Alias, _ *ValueList) string {
	return `a`
}

var testCondition2 = func(_ Driver, _ Alias, _ *ValueList) string {
	return `b`
}

func TestWhere(t *testing.T) {
	b, check := testBuilder(t, false)

	b.Where(testCondition, testCondition2)
	check(true, `WHERE a`+"\n\t"+`AND b`)

	b.Where()
	check(false, ``)
}

func TestHaving(t *testing.T) {
	b, check := testBuilder(t, false)

	b.Having(testCondition, testCondition2)
	check(true, `HAVING a`+"\n\t"+`AND b`)

	b.Having()
	check(false, ``)
}

// Other

func TestGroupBy(t *testing.T) {
	b, check := testBuilder(t, false)

	b.GroupBy(testFieldA, testFieldB)
	check(true, `GROUP BY colA, colB`)

	b.GroupBy()
	check(false, ``)
}

func TestOrderBy(t *testing.T) {
	b, check := testBuilder(t, false)

	b.OrderBy(Asc(testFieldA), Desc(testFieldB))
	check(true, `ORDER BY colA ASC, colB DESC`)

	b.OrderBy()
	check(false, ``)
}

func TestLimit(t *testing.T) {
	b, check := testBuilder(t, false)

	b.Limit(2)
	check(true, `LIMIT 2`)

	b.Limit(0)
	check(false, ``)
}

func TestOffset(t *testing.T) {
	b, check := testBuilder(t, false)

	b.Offset(2)
	check(true, `OFFSET 2`)

	b.Offset(0)
	check(false, ``)
}

func TestValues(t *testing.T) {
	b, check := testBuilder(t, false)

	line := []Field{Value(1), Value(2), Value(3)}

	b.Values([][]Field{line})
	check(true, `VALUES (?, ?, ?)`)

	b.Values([][]Field{line, line})
	check(true, `VALUES`+"\n\t"+
		`(?, ?, ?),`+"\n\t"+
		`(?, ?, ?)`,
	)
}

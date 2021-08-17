package filter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

type TestModel struct {
	Name string
	ID   uint
}

func TestSQLEscape(t *testing.T) {
	tx := &gorm.DB{Config: &gorm.Config{
		Dialector: tests.DummyDialector{},
	}}
	assert.Equal(t, "`name`", SQLEscape(tx, "name"))
}

func TestGetTableName(t *testing.T) {
	tx := &gorm.DB{
		Config:    &gorm.Config{Dialector: tests.DummyDialector{}},
		Statement: &gorm.Statement{},
	}
	tx.Statement.DB = tx

	assert.Empty(t, getTableName(tx))

	tx = tx.Table("users")

	assert.Equal(t, "users.", getTableName(tx))

	tx, _ = gorm.Open(tests.DummyDialector{}, nil)
	tx = tx.Model(&TestModel{})

	assert.Equal(t, "test_models.", getTableName(tx))

	assert.Panics(t, func() {
		tx, _ = gorm.Open(tests.DummyDialector{}, nil)
		tx = tx.Model(1)
		fmt.Println(getTableName(tx))
	})
}

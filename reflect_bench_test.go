package filter

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func BenchmarkParseModel(b *testing.B) {
	identityCache = make(map[string]*modelIdentity, 10)
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		parseModel(db, &TestModel{})
	}
}

func BenchmarkParseModelNoCache(b *testing.B) {
	db, _ := gorm.Open(&tests.DummyDialector{}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		identityCache = make(map[string]*modelIdentity, 10)
		parseModel(db, &TestModel{})
	}
}

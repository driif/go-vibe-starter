package db_test

import (
	"testing"

	"github.com/driif/go-vibe-starter/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestDBTypeConversions(t *testing.T) {
	i := int64(19)
	res := db.NullInt64FromPtr(&i)
	assert.Equal(t, int64(19), res.Int64)
	assert.True(t, res.Valid)

	res = db.NullInt64FromPtr(nil)
	assert.False(t, res.Valid)

	f := 19.9999
	res2 := db.NullFloat64FromPtr(&f)
	assert.Equal(t, 19.9999, res2.Float64)
	assert.True(t, res2.Valid)

	res2 = db.NullFloat64FromPtr(nil)
	assert.False(t, res2.Valid)
}

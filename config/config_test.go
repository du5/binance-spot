package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigInitialization(t *testing.T) {
	timeAt := Config.OrderLists[0].TimeAt
	parsedTime, err := time.Parse("15:04:00", timeAt)
	assert.NoError(t, err)
	assert.Equal(t, 13, parsedTime.Hour())
	assert.Equal(t, 00, parsedTime.Minute())
	assert.Equal(t, 0, parsedTime.Second())

	timeAt = Config.OrderLists[1].TimeAt
	parsedTime, err = time.Parse("15:04:00", timeAt)
	assert.NoError(t, err)
	assert.Equal(t, 15, parsedTime.Hour())
	assert.Equal(t, 30, parsedTime.Minute())
	assert.Equal(t, 0, parsedTime.Second())
}

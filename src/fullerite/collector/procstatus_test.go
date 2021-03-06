package collector

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcStatusConfigureEmptyConfig(t *testing.T) {
	config := make(map[string]interface{})

	ps := NewProcStatus(nil, 123, nil)
	ps.Configure(config)

	assert.Equal(t, 123, ps.Interval())
	assert.Equal(t, "", ps.ProcessName())
}

func TestProcStatusConfigure(t *testing.T) {
	config := make(map[string]interface{})
	config["interval"] = 9999
	config["processName"] = "fullerite"

	dims := map[string]string{
		"currentDirectory": ".*",
	}
	config["generatedDimensions"] = dims

	regex, _ := regexp.Compile(".*")
	compRegex := map[string]*regexp.Regexp{
		"currentDirectory": regex,
	}

	ps := NewProcStatus(nil, 123, nil)
	ps.Configure(config)

	assert.Equal(t, 9999, ps.Interval())
	assert.Equal(t, "fullerite", ps.ProcessName())
	assert.Equal(t, compRegex, ps.compiledRegex)
}

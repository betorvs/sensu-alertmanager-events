package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	v2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
)

func TestCheckArgs(t *testing.T) {
	assert := assert.New(t)
	plugin.AgentAPIURL = "http://127.0.0.1:3031/events"
	event := v2.FixtureEvent("entity1", "check1")
	status, err := checkArgs(event)
	assert.NoError(err)
	assert.Equal(sensu.CheckStateOK, status)
	status, err = checkArgs(event)
	assert.NoError(err)
	assert.Equal(sensu.CheckStateOK, status)
	plugin.AlertmanagerLabelEntity = "cluster"
	plugin.SensuProxyEntity = "k8s-cluster"
	status, err = checkArgs(event)
	assert.Error(err)
	assert.Equal(sensu.CheckStateWarning, status)
}

func TestSubmitEventAgentAPI(t *testing.T) {
	testcases := []struct {
		httpStatus  int
		expectError bool
	}{
		{http.StatusOK, false},
		{http.StatusBadRequest, true},
	}
	for _, tc := range testcases {
		assert := assert.New(t)
		event := v2.FixtureEvent("entity1", "check1")
		var test = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(err)
			eV := &v2.Event{}
			_ = json.Unmarshal(body, eV)
			w.WriteHeader(tc.httpStatus)
		}))
		_, _ = url.ParseRequestURI(test.URL)
		plugin.AgentAPIURL = test.URL
		err := submitEventAgentAPI(event)
		if tc.expectError {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
	}
}

func TestStringInSlice(t *testing.T) {
	testSlice := []string{"foo", "bar", "test"}
	testString := "test"
	testResult := stringInSlice(testString, testSlice)
	assert.True(t, testResult)
}

func TestSearchLabels(t *testing.T) {
	event1 := v2.FixtureEvent("entity1", "check1")
	event1.Labels["testa"] = "valuea"
	event1.Labels["testb"] = "valueb"
	event1.Labels["testc"] = "valuec"
	labels := make(map[string]string)
	res1 := searchLabels(event1, labels)
	assert.False(t, res1)

	labels["testa"] = "valuea"
	labels["testc"] = "valuec"
	res2 := searchLabels(event1, labels)
	assert.True(t, res2)

	excludeLabels := make(map[string]string)
	excludeLabels["testc"] = "valuec"
	res3 := searchLabels(event1, excludeLabels)
	assert.True(t, res3)
}

func TestRemoveSpecialCharacters(t *testing.T) {
	test1 := "[test]Check-Long(testa / testb)"
	res1 := removeSpecialCharacters(test1)
	assert.NotContains(t, res1, "/")
	assert.NotContains(t, res1, "(")
	assert.NotContains(t, res1, "[")
	test2 := "JustCommon-check-http"
	res2 := removeSpecialCharacters(test2)
	assert.Contains(t, res2, "Common-check")
}

func TestParseLabelArg(t *testing.T) {
	test1 := "OneLabel=OneValue"
	val1 := map[string]string{"OneLabel": "OneValue"}
	res1 := parseLabelArg(test1)
	assert.Equal(t, val1, res1)
	test2 := "OneLabel=OneValue,TwoLabel=TwoValue"
	val2 := map[string]string{"OneLabel": "OneValue", "TwoLabel": "TwoValue"}
	res2 := parseLabelArg(test2)
	assert.Equal(t, val2, res2)
}

func TestMergeStringMaps(t *testing.T) {
	left1 := map[string]string{"left1": "leftValue1"}
	right1 := map[string]string{"right1": "rightValue1"}
	val1 := map[string]string{"left1": "leftValue1", "right1": "rightValue1"}
	res1 := mergeStringMaps(left1, right1)
	assert.Equal(t, val1, res1)
	left2 := map[string]string{"left1": "leftValue1"}
	right2 := map[string]string{"right1": "rightValue1", "left1": "rightValueLeft1"}
	val2 := map[string]string{"left1": "leftValue1", "right1": "rightValue1"}
	res2 := mergeStringMaps(left2, right2)
	assert.Equal(t, val2, res2)
}

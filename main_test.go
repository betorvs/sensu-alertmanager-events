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
	test3 := "OneLabelOneValue,TwoLabel=TwoValue"
	val3 := map[string]string{"TwoLabel": "TwoValue"}
	res3 := parseLabelArg(test3)
	assert.Equal(t, val3, res3)
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
	left3 := map[string]string{"left1": "leftValue1"}
	right3 := map[string]string{}
	val3 := map[string]string{"left1": "leftValue1"}
	res3 := mergeStringMaps(left3, right3)
	assert.Equal(t, val3, res3)
	left4 := map[string]string{}
	right4 := map[string]string{"right1": "rightValue1"}
	val4 := map[string]string{"right1": "rightValue1"}
	res4 := mergeStringMaps(left4, right4)
	assert.Equal(t, val4, res4)
}

func TestSplitString(t *testing.T) {
	test1 := "key1=value1"
	res1 := "key1"
	res2 := "value1"
	val1, val2 := splitString(test1, "=")
	assert.Equal(t, res1, val1)
	assert.Equal(t, res2, val2)
	assert.NotEqual(t, res1, val2)
}

func TestMakeRewriteAnnotation(t *testing.T) {
	test1 := "key1=value1,key2=value2"
	res1 := map[string]string{"key1": "value1", "key2": "value2"}
	val1 := makeRewriteAnnotation(test1)
	assert.Equal(t, res1, val1)
	test2 := "key1=value1,key2/subkey2=value2"
	res2 := map[string]string{"key1": "value1", "key2/subkey2": "value2"}
	val2 := makeRewriteAnnotation(test2)
	assert.Equal(t, res2, val2)
	test3 := "key1=value1,key2=value2,"
	res3 := map[string]string{"key1": "value1", "key2": "value2"}
	val3 := makeRewriteAnnotation(test3)
	assert.Equal(t, res3, val3)
}

func TestRewriteAnnotation(t *testing.T) {
	test1 := "opsgenie_priority=sensu.io/plugins/sensu-opsgenie-handler/config/priority"
	val1 := "opsgenie_priority"
	expected1 := "sensu.io/plugins/sensu-opsgenie-handler/config/priority"
	rule1 := makeRewriteAnnotation(test1)
	rule1res := map[string]string{"opsgenie_priority": "sensu.io/plugins/sensu-opsgenie-handler/config/priority"}
	assert.Equal(t, rule1, rule1res)
	res1, err1 := rewriteAnnotation(val1, rule1)
	assert.NoError(t, err1)
	assert.Equal(t, res1, expected1)
}

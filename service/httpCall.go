package service

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/kotanetes/go-test-it/model"
	"github.com/sirupsen/logrus"
)

const (
	// GraphQL - constant used to check test type
	GraphQL = "graphql"
)

var client remoteClient

type remoteClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// InitHTTPClient - intializes httpClient
// this can be used to mock the http calls for this project
func InitHTTPClient(c remoteClient) {
	client = c
}

func doCall(test model.TestScenario) ([]byte, int) {

	var (
		reqBody     []byte
		err         error
		requestBody io.Reader
		body        interface{}
	)

	if test.Body != nil {
		switch test.Type {
		case GraphQL:
			bodyMap := make(map[string]interface{})
			bodyMap["query"] = test.Body
			body = bodyMap
		default:
			body = test.Body
		}
		reqBody, err = json.Marshal(body)
		if err != nil {
			logrus.Error(err)
		}

		requestBody = strings.NewReader(string(reqBody))
	}

	req, err := http.NewRequest(test.Method, test.URL, requestBody)
	if err != nil {
		logrus.Error(err)
	}

	if test.Header.Authorization != "" {
		req.Header.Add("authorization", test.Header.Authorization)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
	}

	return bodyBytes, resp.StatusCode
}

// MakeHTTPCall - performs an http call
func MakeHTTPCall(scenarios []model.TestScenario, scenarioName string) (map[string]bool, int) {
	var (
		err     error
		ignored int
	)

	finalResult := make(map[string]bool)
	testScenarios := make(map[string]model.TestScenario, 0)

	for _, scenario := range scenarios {
		if !scenario.Ignore {
			// add not ignored scenarios to the map and excute that test
			testScenarios[scenario.Scenario] = scenario
		} else {
			logrus.Info(fmt.Sprintf("Test %v Ignored", scenario.Scenario))
			ignored++
		}
	}

	for _, test := range testScenarios {

		result := make(map[string]interface{})

		bodyBytes, statusCode := doCall(test)

		err = json.Unmarshal(bodyBytes, &result)
		if err != nil {
			logrus.Error(err)
		}

		switch {
		case statusCode != test.ExpectedStatusCode:
			logrus.Info(fmt.Sprintf("Test %v failed, expected status %v got %v", test.Scenario, test.ExpectedStatusCode, statusCode))
			finalResult[test.Scenario] = false
		case !reflect.DeepEqual(result, test.ExpectedResult):
			fmt.Printf("expected:%v,got: %v\n", test.ExpectedResult, result)
			logrus.Info(fmt.Sprintf("Test %v failed, retunred response is not as expected", test.Scenario))
			finalResult[test.Scenario] = false
		default:
			logrus.Info(fmt.Sprintf("Test %v Passed", test.Scenario))
			finalResult[test.Scenario] = true
		}
	}
	return finalResult, ignored
}

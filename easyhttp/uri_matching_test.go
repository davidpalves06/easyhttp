package easyhttp

import "testing"

type UriMatchTest struct {
	requestPath    string
	pattern        string
	expectedResult bool
}

var uriMatchTests = []UriMatchTest{
	{requestPath: "/path", pattern: "/*", expectedResult: true},
	{requestPath: "/", pattern: "*", expectedResult: true},
	{requestPath: "/path", pattern: "/", expectedResult: false},
	{requestPath: "/path", pattern: "/path/resource", expectedResult: false},
	{requestPath: "/path/resource", pattern: "/*", expectedResult: true},
	{requestPath: "/path/resource", pattern: "/path/*", expectedResult: true},
	{requestPath: "/path/resource", pattern: "/path/resource/*", expectedResult: false},
	{requestPath: "/path/resource", pattern: "/path/resource", expectedResult: true},
	{requestPath: "/path/resource", pattern: "/path/resource/", expectedResult: true},
	{requestPath: "/path/resource/local", pattern: "/path/*/local", expectedResult: true},
	{requestPath: "/path/resource/local", pattern: "/path/*/test", expectedResult: false},
	{requestPath: "/path/resource/local/test", pattern: "/path/*/test", expectedResult: true},
	{requestPath: "/path/resource/local/test", pattern: "/path/*/local/test", expectedResult: true},
	{requestPath: "/path/resource/local/test", pattern: "/path/*/test/test", expectedResult: false},
	{requestPath: "/path/resource/local/test", pattern: "/path/resource/*/local/test", expectedResult: false},
}

func TestUriMatching(t *testing.T) {
	for _, test := range uriMatchTests {
		got := isURIMatch(test.requestPath, test.pattern)
		if got != test.expectedResult {
			t.Errorf("Test failed. Request: %s;Pattern: %s; Excepted: %v; Got: %v\n", test.requestPath, test.pattern, test.expectedResult, got)
		}
	}
}

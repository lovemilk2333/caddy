// Copyright 2015 Matthew Holt and The Caddy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package caddyhttp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"testing"

	"github.com/lovemilk2333/caddy/v2"
)

func TestHostMatcher(t *testing.T) {
	err := os.Setenv("GO_BENCHMARK_DOMAIN", "localhost")
	if err != nil {
		t.Errorf("error while setting up environment: %v", err)
	}

	for i, tc := range []struct {
		match  MatchHost
		input  string
		expect bool
	}{
		{
			match:  MatchHost{},
			input:  "example.com",
			expect: false,
		},
		{
			match:  MatchHost{"example.com"},
			input:  "example.com",
			expect: true,
		},
		{
			match:  MatchHost{"EXAMPLE.COM"},
			input:  "example.com",
			expect: true,
		},
		{
			match:  MatchHost{"example.com"},
			input:  "EXAMPLE.COM",
			expect: true,
		},
		{
			match:  MatchHost{"example.com"},
			input:  "foo.example.com",
			expect: false,
		},
		{
			match:  MatchHost{"example.com"},
			input:  "EXAMPLE.COM",
			expect: true,
		},
		{
			match:  MatchHost{"foo.example.com"},
			input:  "foo.example.com",
			expect: true,
		},
		{
			match:  MatchHost{"foo.example.com"},
			input:  "bar.example.com",
			expect: false,
		},
		{
			match:  MatchHost{"éxàmplê.com"},
			input:  "xn--xmpl-0na6cm.com",
			expect: true,
		},
		{
			match:  MatchHost{"*.example.com"},
			input:  "example.com",
			expect: false,
		},
		{
			match:  MatchHost{"*.example.com"},
			input:  "SUB.EXAMPLE.COM",
			expect: true,
		},
		{
			match:  MatchHost{"*.example.com"},
			input:  "foo.example.com",
			expect: true,
		},
		{
			match:  MatchHost{"*.example.com"},
			input:  "foo.bar.example.com",
			expect: false,
		},
		{
			match:  MatchHost{"*.example.com", "example.net"},
			input:  "example.net",
			expect: true,
		},
		{
			match:  MatchHost{"example.net", "*.example.com"},
			input:  "foo.example.com",
			expect: true,
		},
		{
			match:  MatchHost{"*.example.net", "*.*.example.com"},
			input:  "foo.bar.example.com",
			expect: true,
		},
		{
			match:  MatchHost{"*.example.net", "sub.*.example.com"},
			input:  "sub.foo.example.com",
			expect: true,
		},
		{
			match:  MatchHost{"*.example.net", "sub.*.example.com"},
			input:  "sub.foo.example.net",
			expect: false,
		},
		{
			match:  MatchHost{"www.*.*"},
			input:  "www.example.com",
			expect: true,
		},
		{
			match:  MatchHost{"example.com"},
			input:  "example.com:5555",
			expect: true,
		},
		{
			match:  MatchHost{"{env.GO_BENCHMARK_DOMAIN}"},
			input:  "localhost",
			expect: true,
		},
		{
			match:  MatchHost{"{env.GO_NONEXISTENT}"},
			input:  "localhost",
			expect: false,
		},
	} {
		req := &http.Request{Host: tc.input}
		repl := caddy.NewReplacer()
		ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
		req = req.WithContext(ctx)

		if err := tc.match.Provision(caddy.Context{}); err != nil {
			t.Errorf("Test %d %v: provisioning failed: %v", i, tc.match, err)
		}

		actual, err := tc.match.MatchWithError(req)
		if err != nil {
			t.Errorf("Test %d %v: matching failed: %v", i, tc.match, err)
		}
		if actual != tc.expect {
			t.Errorf("Test %d %v: Expected %t, got %t for '%s'", i, tc.match, tc.expect, actual, tc.input)
			continue
		}
	}
}

func TestPathMatcher(t *testing.T) {
	for i, tc := range []struct {
		match        MatchPath // not URI-encoded because not parsing from a URI
		input        string    // should be valid URI encoding (escaped) since it will become part of a request
		expect       bool
		provisionErr bool
	}{
		{
			match:  MatchPath{},
			input:  "/",
			expect: false,
		},
		{
			match:  MatchPath{"/"},
			input:  "/",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/bar"},
			input:  "/",
			expect: false,
		},
		{
			match:  MatchPath{"/foo/bar"},
			input:  "/foo/bar",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/bar/"},
			input:  "/foo/bar",
			expect: false,
		},
		{
			match:  MatchPath{"/foo/bar/"},
			input:  "/foo/bar/",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/bar/", "/other"},
			input:  "/other/",
			expect: false,
		},
		{
			match:  MatchPath{"/foo/bar/", "/other"},
			input:  "/other",
			expect: true,
		},
		{
			match:  MatchPath{"*.ext"},
			input:  "/foo/bar.ext",
			expect: true,
		},
		{
			match:  MatchPath{"*.php"},
			input:  "/index.PHP",
			expect: true,
		},
		{
			match:  MatchPath{"*.ext"},
			input:  "/foo/bar.ext",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/*/baz"},
			input:  "/foo/bar/baz",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/*/baz/bam"},
			input:  "/foo/bar/bam",
			expect: false,
		},
		{
			match:  MatchPath{"*substring*"},
			input:  "/foo/substring/bar.txt",
			expect: true,
		},
		{
			match:  MatchPath{"/foo"},
			input:  "/foo/bar",
			expect: false,
		},
		{
			match:  MatchPath{"/foo"},
			input:  "/foo/bar",
			expect: false,
		},
		{
			match:  MatchPath{"/foo"},
			input:  "/FOO",
			expect: true,
		},
		{
			match:  MatchPath{"/foo*"},
			input:  "/FOOOO",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/bar.txt"},
			input:  "/foo/BAR.txt",
			expect: true,
		},
		{
			match:  MatchPath{"/foo*"},
			input:  "//foo/bar",
			expect: true,
		},
		{
			match:  MatchPath{"/foo"},
			input:  "//foo",
			expect: true,
		},
		{
			match:  MatchPath{"//foo"},
			input:  "/foo",
			expect: false,
		},
		{
			match:  MatchPath{"//foo"},
			input:  "//foo",
			expect: true,
		},
		{
			match:  MatchPath{"/foo//*"},
			input:  "/foo//bar",
			expect: true,
		},
		{
			match:  MatchPath{"/foo//*"},
			input:  "/foo/%2Fbar",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/%2F*"},
			input:  "/foo/%2Fbar",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/%2F*"},
			input:  "/foo//bar",
			expect: false,
		},
		{
			match:  MatchPath{"/foo//bar"},
			input:  "/foo//bar",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/*//bar"},
			input:  "/foo///bar",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/%*//bar"},
			input:  "/foo///bar",
			expect: true,
		},
		{
			match:  MatchPath{"/foo/%*//bar"},
			input:  "/foo//%2Fbar",
			expect: true,
		},
		{
			match:  MatchPath{"/foo*"},
			input:  "/%2F/foo",
			expect: true,
		},
		{
			match:  MatchPath{"*"},
			input:  "/",
			expect: true,
		},
		{
			match:  MatchPath{"*"},
			input:  "/foo/bar",
			expect: true,
		},
		{
			match:  MatchPath{"**"},
			input:  "/",
			expect: true,
		},
		{
			match:  MatchPath{"**"},
			input:  "/foo/bar",
			expect: true,
		},
		// notice these next three test cases are the same normalized path but are written differently
		{
			match:  MatchPath{"/%25@.txt"},
			input:  "/%25@.txt",
			expect: true,
		},
		{
			match:  MatchPath{"/%25@.txt"},
			input:  "/%25%40.txt",
			expect: true,
		},
		{
			match:  MatchPath{"/%25%40.txt"},
			input:  "/%25%40.txt",
			expect: true,
		},
		{
			match:  MatchPath{"/bands/*/*"},
			input:  "/bands/AC%2FDC/T.N.T",
			expect: false, // because * operates in normalized space
		},
		{
			match:  MatchPath{"/bands/%*/%*"},
			input:  "/bands/AC%2FDC/T.N.T",
			expect: true,
		},
		{
			match:  MatchPath{"/bands/%*/%*"},
			input:  "/bands/AC/DC/T.N.T",
			expect: false,
		},
		{
			match:  MatchPath{"/bands/%*"},
			input:  "/bands/AC/DC",
			expect: false, // not a suffix match
		},
		{
			match:  MatchPath{"/bands/%*"},
			input:  "/bands/AC%2FDC",
			expect: true,
		},
		{
			match:  MatchPath{"/foo%2fbar/baz"},
			input:  "/foo%2Fbar/baz",
			expect: true,
		},
		{
			match:  MatchPath{"/foo%2fbar/baz"},
			input:  "/foo/bar/baz",
			expect: false,
		},
		{
			match:  MatchPath{"/foo/bar/baz"},
			input:  "/foo%2fbar/baz",
			expect: true,
		},
	} {
		err := tc.match.Provision(caddy.Context{})
		if err == nil && tc.provisionErr {
			t.Errorf("Test %d %v: Expected error provisioning, but there was no error", i, tc.match)
		}
		if err != nil && !tc.provisionErr {
			t.Errorf("Test %d %v: Expected no error provisioning, but there was an error: %v", i, tc.match, err)
		}
		if tc.provisionErr {
			continue // if it's not supposed to provision properly, pointless to test it
		}

		u, err := url.ParseRequestURI(tc.input)
		if err != nil {
			t.Fatalf("Test %d (%v): Invalid request URI (should be rejected by Go's HTTP server): %v", i, tc.input, err)
		}
		req := &http.Request{URL: u}
		repl := caddy.NewReplacer()
		ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
		req = req.WithContext(ctx)

		actual, err := tc.match.MatchWithError(req)
		if err != nil {
			t.Errorf("Test %d %v: matching failed: %v", i, tc.match, err)
		}
		if actual != tc.expect {
			t.Errorf("Test %d %v: Expected %t, got %t for '%s'", i, tc.match, tc.expect, actual, tc.input)
			continue
		}
	}
}

func TestPathMatcherWindows(t *testing.T) {
	// only Windows has this bug where it will ignore
	// trailing dots and spaces in a filename
	if runtime.GOOS != "windows" {
		return
	}

	req := &http.Request{URL: &url.URL{Path: "/index.php . . .."}}
	repl := caddy.NewReplacer()
	ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
	req = req.WithContext(ctx)

	match := MatchPath{"*.php"}
	matched, err := match.MatchWithError(req)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if !matched {
		t.Errorf("Expected to match; should ignore trailing dots and spaces")
	}
}

func TestPathREMatcher(t *testing.T) {
	for i, tc := range []struct {
		match      MatchPathRE
		input      string
		expect     bool
		expectRepl map[string]string
	}{
		{
			match:  MatchPathRE{},
			input:  "/",
			expect: true,
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "/"}},
			input:  "/",
			expect: true,
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "^/foo"}},
			input:  "/foo",
			expect: true,
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "^/foo"}},
			input:  "/foo/",
			expect: true,
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "^/foo"}},
			input:  "//foo",
			expect: true,
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "^/foo"}},
			input:  "//foo/",
			expect: true,
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "^/foo"}},
			input:  "/%2F/foo/",
			expect: true,
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "/bar"}},
			input:  "/foo/",
			expect: false,
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "^/bar"}},
			input:  "/foo/bar",
			expect: false,
		},
		{
			match:      MatchPathRE{MatchRegexp{Pattern: "^/foo/(.*)/baz$", Name: "name"}},
			input:      "/foo/bar/baz",
			expect:     true,
			expectRepl: map[string]string{"name.1": "bar"},
		},
		{
			match:      MatchPathRE{MatchRegexp{Pattern: "^/foo/(?P<myparam>.*)/baz$", Name: "name"}},
			input:      "/foo/bar/baz",
			expect:     true,
			expectRepl: map[string]string{"name.myparam": "bar"},
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "^/%@.txt"}},
			input:  "/%25@.txt",
			expect: true,
		},
		{
			match:  MatchPathRE{MatchRegexp{Pattern: "^/%25@.txt"}},
			input:  "/%25@.txt",
			expect: false,
		},
	} {
		// compile the regexp and validate its name
		err := tc.match.Provision(caddy.Context{})
		if err != nil {
			t.Errorf("Test %d %v: Provisioning: %v", i, tc.match, err)
			continue
		}
		err = tc.match.Validate()
		if err != nil {
			t.Errorf("Test %d %v: Validating: %v", i, tc.match, err)
			continue
		}

		// set up the fake request and its Replacer
		u, err := url.ParseRequestURI(tc.input)
		if err != nil {
			t.Fatalf("Test %d: Bad input URI: %v", i, err)
		}
		req := &http.Request{URL: u}
		repl := caddy.NewReplacer()
		ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
		req = req.WithContext(ctx)
		addHTTPVarsToReplacer(repl, req, httptest.NewRecorder())

		actual, err := tc.match.MatchWithError(req)
		if err != nil {
			t.Errorf("Test %d %v: matching failed: %v", i, tc.match, err)
		}
		if actual != tc.expect {
			t.Errorf("Test %d [%v]: Expected %t, got %t for input '%s'",
				i, tc.match.Pattern, tc.expect, actual, tc.input)
			continue
		}

		for key, expectVal := range tc.expectRepl {
			placeholder := fmt.Sprintf("{http.regexp.%s}", key)
			actualVal := repl.ReplaceAll(placeholder, "<empty>")
			if actualVal != expectVal {
				t.Errorf("Test %d [%v]: Expected placeholder {http.regexp.%s} to be '%s' but got '%s'",
					i, tc.match.Pattern, key, expectVal, actualVal)
				continue
			}
		}
	}
}

func TestHeaderMatcher(t *testing.T) {
	repl := caddy.NewReplacer()
	repl.Set("a", "foobar")

	for i, tc := range []struct {
		match  MatchHeader
		input  http.Header // make sure these are canonical cased (std lib will do that in a real request)
		host   string
		expect bool
	}{
		{
			match:  MatchHeader{"Field": []string{"foo"}},
			input:  http.Header{"Field": []string{"foo"}},
			expect: true,
		},
		{
			match:  MatchHeader{"Field": []string{"foo", "bar"}},
			input:  http.Header{"Field": []string{"bar"}},
			expect: true,
		},
		{
			match:  MatchHeader{"Field": []string{"foo", "bar"}},
			input:  http.Header{"Alakazam": []string{"kapow"}},
			expect: false,
		},
		{
			match:  MatchHeader{"Field": []string{"foo", "bar"}},
			input:  http.Header{"Field": []string{"kapow"}},
			expect: false,
		},
		{
			match:  MatchHeader{"Field": []string{"foo", "bar"}},
			input:  http.Header{"Field": []string{"kapow", "foo"}},
			expect: true,
		},
		{
			match:  MatchHeader{"Field1": []string{"foo"}, "Field2": []string{"bar"}},
			input:  http.Header{"Field1": []string{"foo"}, "Field2": []string{"bar"}},
			expect: true,
		},
		{
			match:  MatchHeader{"field1": []string{"foo"}, "field2": []string{"bar"}},
			input:  http.Header{"Field1": []string{"foo"}, "Field2": []string{"bar"}},
			expect: true,
		},
		{
			match:  MatchHeader{"field1": []string{"foo"}, "field2": []string{"bar"}},
			input:  http.Header{"Field1": []string{"foo"}, "Field2": []string{"kapow"}},
			expect: false,
		},
		{
			match:  MatchHeader{"field1": []string{"*"}},
			input:  http.Header{"Field1": []string{"foo"}},
			expect: true,
		},
		{
			match:  MatchHeader{"field1": []string{"*"}},
			input:  http.Header{"Field2": []string{"foo"}},
			expect: false,
		},
		{
			match:  MatchHeader{"Field1": []string{"foo*"}},
			input:  http.Header{"Field1": []string{"foo"}},
			expect: true,
		},
		{
			match:  MatchHeader{"Field1": []string{"foo*"}},
			input:  http.Header{"Field1": []string{"asdf", "foobar"}},
			expect: true,
		},
		{
			match:  MatchHeader{"Field1": []string{"*bar"}},
			input:  http.Header{"Field1": []string{"asdf", "foobar"}},
			expect: true,
		},
		{
			match:  MatchHeader{"host": []string{"localhost"}},
			input:  http.Header{},
			host:   "localhost",
			expect: true,
		},
		{
			match:  MatchHeader{"host": []string{"localhost"}},
			input:  http.Header{},
			host:   "caddyserver.com",
			expect: false,
		},
		{
			match:  MatchHeader{"Must-Not-Exist": nil},
			input:  http.Header{},
			expect: true,
		},
		{
			match:  MatchHeader{"Must-Not-Exist": nil},
			input:  http.Header{"Must-Not-Exist": []string{"do not match"}},
			expect: false,
		},
		{
			match:  MatchHeader{"Foo": []string{"{a}"}},
			input:  http.Header{"Foo": []string{"foobar"}},
			expect: true,
		},
		{
			match:  MatchHeader{"Foo": []string{"{a}"}},
			input:  http.Header{"Foo": []string{"asdf"}},
			expect: false,
		},
		{
			match:  MatchHeader{"Foo": []string{"{a}*"}},
			input:  http.Header{"Foo": []string{"foobar-baz"}},
			expect: true,
		},
	} {
		req := &http.Request{Header: tc.input, Host: tc.host}
		ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
		req = req.WithContext(ctx)

		actual, err := tc.match.MatchWithError(req)
		if err != nil {
			t.Errorf("Test %d %v: matching failed: %v", i, tc.match, err)
		}
		if actual != tc.expect {
			t.Errorf("Test %d %v: Expected %t, got %t for '%s'", i, tc.match, tc.expect, actual, tc.input)
			continue
		}
	}
}

func TestQueryMatcher(t *testing.T) {
	for i, tc := range []struct {
		scenario string
		match    MatchQuery
		input    string
		expect   bool
	}{
		{
			scenario: "non match against a specific value",
			match:    MatchQuery{"debug": []string{"1"}},
			input:    "/",
			expect:   false,
		},
		{
			scenario: "match against a specific value",
			match:    MatchQuery{"debug": []string{"1"}},
			input:    "/?debug=1",
			expect:   true,
		},
		{
			scenario: "match against a wildcard",
			match:    MatchQuery{"debug": []string{"*"}},
			input:    "/?debug=something",
			expect:   true,
		},
		{
			scenario: "non match against a wildcarded",
			match:    MatchQuery{"debug": []string{"*"}},
			input:    "/?other=something",
			expect:   false,
		},
		{
			scenario: "match against an empty value",
			match:    MatchQuery{"debug": []string{""}},
			input:    "/?debug",
			expect:   true,
		},
		{
			scenario: "non match against an empty value",
			match:    MatchQuery{"debug": []string{""}},
			input:    "/?someparam",
			expect:   false,
		},
		{
			scenario: "empty matcher value should match empty query",
			match:    MatchQuery{},
			input:    "/?",
			expect:   true,
		},
		{
			scenario: "nil matcher value should NOT match a non-empty query",
			match:    MatchQuery{},
			input:    "/?foo=bar",
			expect:   false,
		},
		{
			scenario: "non-nil matcher should NOT match an empty query",
			match:    MatchQuery{"": nil},
			input:    "/?",
			expect:   false,
		},
		{
			scenario: "match against a placeholder value",
			match:    MatchQuery{"debug": []string{"{http.vars.debug}"}},
			input:    "/?debug=1",
			expect:   true,
		},
		{
			scenario: "match against a placeholder key",
			match:    MatchQuery{"{http.vars.key}": []string{"1"}},
			input:    "/?somekey=1",
			expect:   true,
		},
		{
			scenario: "do not match when not all query params are present",
			match:    MatchQuery{"debug": []string{"1"}, "foo": []string{"bar"}},
			input:    "/?debug=1",
			expect:   false,
		},
		{
			scenario: "match when all query params are present",
			match:    MatchQuery{"debug": []string{"1"}, "foo": []string{"bar"}},
			input:    "/?debug=1&foo=bar",
			expect:   true,
		},
		{
			scenario: "do not match when the value of a query param does not match",
			match:    MatchQuery{"debug": []string{"1"}, "foo": []string{"bar"}},
			input:    "/?debug=2&foo=bar",
			expect:   false,
		},
		{
			scenario: "do not match when all the values the query params do not match",
			match:    MatchQuery{"debug": []string{"1"}, "foo": []string{"bar"}},
			input:    "/?debug=2&foo=baz",
			expect:   false,
		},
		{
			scenario: "match against two values for the same key",
			match:    MatchQuery{"debug": []string{"1"}},
			input:    "/?debug=1&debug=2",
			expect:   true,
		},
		{
			scenario: "match against two values for the same key",
			match:    MatchQuery{"debug": []string{"2", "1"}},
			input:    "/?debug=2&debug=1",
			expect:   true,
		},
	} {

		u, _ := url.Parse(tc.input)

		req := &http.Request{URL: u}
		repl := caddy.NewReplacer()
		ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
		repl.Set("http.vars.debug", "1")
		repl.Set("http.vars.key", "somekey")
		req = req.WithContext(ctx)
		actual, err := tc.match.MatchWithError(req)
		if err != nil {
			t.Errorf("Test %d %v: matching failed: %v", i, tc.match, err)
		}
		if actual != tc.expect {
			t.Errorf("Test %d %v: Expected %t, got %t for '%s'", i, tc.match, tc.expect, actual, tc.input)
			continue
		}
	}
}

func TestHeaderREMatcher(t *testing.T) {
	for i, tc := range []struct {
		match      MatchHeaderRE
		input      http.Header // make sure these are canonical cased (std lib will do that in a real request)
		host       string
		expect     bool
		expectRepl map[string]string
	}{
		{
			match:  MatchHeaderRE{"Field": &MatchRegexp{Pattern: "foo"}},
			input:  http.Header{"Field": []string{"foo"}},
			expect: true,
		},
		{
			match:  MatchHeaderRE{"Field": &MatchRegexp{Pattern: "$foo^"}},
			input:  http.Header{"Field": []string{"foobar"}},
			expect: false,
		},
		{
			match:      MatchHeaderRE{"Field": &MatchRegexp{Pattern: "^foo(.*)$", Name: "name"}},
			input:      http.Header{"Field": []string{"foobar"}},
			expect:     true,
			expectRepl: map[string]string{"name.1": "bar"},
		},
		{
			match:  MatchHeaderRE{"Field": &MatchRegexp{Pattern: "^foo.*$", Name: "name"}},
			input:  http.Header{"Field": []string{"barfoo", "foobar"}},
			expect: true,
		},
		{
			match:  MatchHeaderRE{"host": &MatchRegexp{Pattern: "^localhost$", Name: "name"}},
			input:  http.Header{},
			host:   "localhost",
			expect: true,
		},
		{
			match:  MatchHeaderRE{"host": &MatchRegexp{Pattern: "^local$", Name: "name"}},
			input:  http.Header{},
			host:   "localhost",
			expect: false,
		},
	} {
		// compile the regexp and validate its name
		err := tc.match.Provision(caddy.Context{})
		if err != nil {
			t.Errorf("Test %d %v: Provisioning: %v", i, tc.match, err)
			continue
		}
		err = tc.match.Validate()
		if err != nil {
			t.Errorf("Test %d %v: Validating: %v", i, tc.match, err)
			continue
		}

		// set up the fake request and its Replacer
		req := &http.Request{Header: tc.input, URL: new(url.URL), Host: tc.host}
		repl := caddy.NewReplacer()
		ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
		req = req.WithContext(ctx)
		addHTTPVarsToReplacer(repl, req, httptest.NewRecorder())

		actual, err := tc.match.MatchWithError(req)
		if err != nil {
			t.Errorf("Test %d %v: matching failed: %v", i, tc.match, err)
		}
		if actual != tc.expect {
			t.Errorf("Test %d [%v]: Expected %t, got %t for input '%s'",
				i, tc.match, tc.expect, actual, tc.input)
			continue
		}

		for key, expectVal := range tc.expectRepl {
			placeholder := fmt.Sprintf("{http.regexp.%s}", key)
			actualVal := repl.ReplaceAll(placeholder, "<empty>")
			if actualVal != expectVal {
				t.Errorf("Test %d [%v]: Expected placeholder {http.regexp.%s} to be '%s' but got '%s'",
					i, tc.match, key, expectVal, actualVal)
				continue
			}
		}
	}
}

func BenchmarkHeaderREMatcher(b *testing.B) {
	i := 0
	match := MatchHeaderRE{"Field": &MatchRegexp{Pattern: "^foo(.*)$", Name: "name"}}
	input := http.Header{"Field": []string{"foobar"}}
	var host string
	err := match.Provision(caddy.Context{})
	if err != nil {
		b.Errorf("Test %d %v: Provisioning: %v", i, match, err)
	}
	err = match.Validate()
	if err != nil {
		b.Errorf("Test %d %v: Validating: %v", i, match, err)
	}

	// set up the fake request and its Replacer
	req := &http.Request{Header: input, URL: new(url.URL), Host: host}
	repl := caddy.NewReplacer()
	ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
	req = req.WithContext(ctx)
	addHTTPVarsToReplacer(repl, req, httptest.NewRecorder())
	for run := 0; run < b.N; run++ {
		match.MatchWithError(req)
	}
}

func TestVarREMatcher(t *testing.T) {
	for i, tc := range []struct {
		desc       string
		match      MatchVarsRE
		input      VarsMiddleware
		expect     bool
		expectRepl map[string]string
	}{
		{
			desc:   "match static value within var set by the VarsMiddleware succeeds",
			match:  MatchVarsRE{"Var1": &MatchRegexp{Pattern: "foo"}},
			input:  VarsMiddleware{"Var1": "here is foo val"},
			expect: true,
		},
		{
			desc:   "value set by VarsMiddleware not satisfying regexp matcher fails to match",
			match:  MatchVarsRE{"Var1": &MatchRegexp{Pattern: "$foo^"}},
			input:  VarsMiddleware{"Var1": "foobar"},
			expect: false,
		},
		{
			desc:       "successfully matched value is captured and its placeholder is added to replacer",
			match:      MatchVarsRE{"Var1": &MatchRegexp{Pattern: "^foo(.*)$", Name: "name"}},
			input:      VarsMiddleware{"Var1": "foobar"},
			expect:     true,
			expectRepl: map[string]string{"name.1": "bar"},
		},
		{
			desc:   "matching against a value of standard variables succeeds",
			match:  MatchVarsRE{"{http.request.method}": &MatchRegexp{Pattern: "^G.[tT]$"}},
			input:  VarsMiddleware{},
			expect: true,
		},
		{
			desc:   "matching against value of var set by the VarsMiddleware and referenced by its placeholder succeeds",
			match:  MatchVarsRE{"{http.vars.Var1}": &MatchRegexp{Pattern: "[vV]ar[0-9]"}},
			input:  VarsMiddleware{"Var1": "var1Value"},
			expect: true,
		},
	} {
		i := i   // capture range value
		tc := tc // capture range value
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			// compile the regexp and validate its name
			err := tc.match.Provision(caddy.Context{})
			if err != nil {
				t.Errorf("Test %d %v: Provisioning: %v", i, tc.match, err)
				return
			}
			err = tc.match.Validate()
			if err != nil {
				t.Errorf("Test %d %v: Validating: %v", i, tc.match, err)
				return
			}

			// set up the fake request and its Replacer
			req := &http.Request{URL: new(url.URL), Method: http.MethodGet}
			repl := caddy.NewReplacer()
			ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
			ctx = context.WithValue(ctx, VarsCtxKey, make(map[string]any))
			req = req.WithContext(ctx)

			addHTTPVarsToReplacer(repl, req, httptest.NewRecorder())

			tc.input.ServeHTTP(httptest.NewRecorder(), req, emptyHandler)

			actual, err := tc.match.MatchWithError(req)
			if err != nil {
				t.Errorf("Test %d %v: matching failed: %v", i, tc.match, err)
			}
			if actual != tc.expect {
				t.Errorf("Test %d [%v]: Expected %t, got %t for input '%s'",
					i, tc.match, tc.expect, actual, tc.input)
				return
			}

			for key, expectVal := range tc.expectRepl {
				placeholder := fmt.Sprintf("{http.regexp.%s}", key)
				actualVal := repl.ReplaceAll(placeholder, "<empty>")
				if actualVal != expectVal {
					t.Errorf("Test %d [%v]: Expected placeholder {http.regexp.%s} to be '%s' but got '%s'",
						i, tc.match, key, expectVal, actualVal)
					return
				}
			}
		})
	}
}

func TestNotMatcher(t *testing.T) {
	for i, tc := range []struct {
		host, path string
		match      MatchNot
		expect     bool
	}{
		{
			host: "example.com", path: "/",
			match:  MatchNot{},
			expect: true,
		},
		{
			host: "example.com", path: "/foo",
			match: MatchNot{
				MatcherSets: []MatcherSet{
					{
						MatchPath{"/foo"},
					},
				},
			},
			expect: false,
		},
		{
			host: "example.com", path: "/bar",
			match: MatchNot{
				MatcherSets: []MatcherSet{
					{
						MatchPath{"/foo"},
					},
				},
			},
			expect: true,
		},
		{
			host: "example.com", path: "/bar",
			match: MatchNot{
				MatcherSets: []MatcherSet{
					{
						MatchPath{"/foo"},
					},
					{
						MatchHost{"example.com"},
					},
				},
			},
			expect: false,
		},
		{
			host: "example.com", path: "/bar",
			match: MatchNot{
				MatcherSets: []MatcherSet{
					{
						MatchPath{"/bar"},
					},
					{
						MatchHost{"example.com"},
					},
				},
			},
			expect: false,
		},
		{
			host: "example.com", path: "/foo",
			match: MatchNot{
				MatcherSets: []MatcherSet{
					{
						MatchPath{"/bar"},
					},
					{
						MatchHost{"sub.example.com"},
					},
				},
			},
			expect: true,
		},
		{
			host: "example.com", path: "/foo",
			match: MatchNot{
				MatcherSets: []MatcherSet{
					{
						MatchPath{"/foo"},
						MatchHost{"example.com"},
					},
				},
			},
			expect: false,
		},
		{
			host: "example.com", path: "/foo",
			match: MatchNot{
				MatcherSets: []MatcherSet{
					{
						MatchPath{"/bar"},
						MatchHost{"example.com"},
					},
				},
			},
			expect: true,
		},
	} {
		req := &http.Request{Host: tc.host, URL: &url.URL{Path: tc.path}}
		repl := caddy.NewReplacer()
		ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
		req = req.WithContext(ctx)

		actual, err := tc.match.MatchWithError(req)
		if err != nil {
			t.Errorf("Test %d %v: matching failed: %v", i, tc.match, err)
		}
		if actual != tc.expect {
			t.Errorf("Test %d %+v: Expected %t, got %t for: host=%s path=%s'", i, tc.match, tc.expect, actual, tc.host, tc.path)
			continue
		}
	}
}

func BenchmarkLargeHostMatcher(b *testing.B) {
	// this benchmark simulates a large host matcher (thousands of entries) where each
	// value is an exact hostname (not a placeholder or wildcard) - compare the results
	// of this with and without the binary search (comment out the various fast path
	// sections in Match) to conduct experiments

	const n = 10000
	lastHost := fmt.Sprintf("%d.example.com", n-1)
	req := &http.Request{Host: lastHost}
	repl := caddy.NewReplacer()
	ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
	req = req.WithContext(ctx)

	matcher := make(MatchHost, n)
	for i := 0; i < n; i++ {
		matcher[i] = fmt.Sprintf("%d.example.com", i)
	}
	err := matcher.Provision(caddy.Context{})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.MatchWithError(req)
	}
}

func BenchmarkHostMatcherWithoutPlaceholder(b *testing.B) {
	req := &http.Request{Host: "localhost"}
	repl := caddy.NewReplacer()
	ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
	req = req.WithContext(ctx)

	match := MatchHost{"localhost"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		match.MatchWithError(req)
	}
}

func BenchmarkHostMatcherWithPlaceholder(b *testing.B) {
	err := os.Setenv("GO_BENCHMARK_DOMAIN", "localhost")
	if err != nil {
		b.Errorf("error while setting up environment: %v", err)
	}

	req := &http.Request{Host: "localhost"}
	repl := caddy.NewReplacer()
	ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
	req = req.WithContext(ctx)
	match := MatchHost{"{env.GO_BENCHMARK_DOMAIN}"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		match.MatchWithError(req)
	}
}

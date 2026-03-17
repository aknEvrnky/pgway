package domain

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchType_IsValid(t *testing.T) {
	for _, tt := range []struct {
		name     string
		mt       MatchType
		expected bool
	}{
		{"host is valid", MatchTypeHost, true},
		{"host_suffix is valid", MatchTypeHostSuffix, true},
		{"path_prefix is valid", MatchTypePathPrefix, true},
		{"path_regex is valid", MatchTypePathRegex, true},
		{"method is valid", MatchTypeMethod, true},
		{"header is valid", MatchTypeHeader, true},
		{"catch_all is valid", MatchTypeCatchAll, true},
		{"empty is invalid", MatchType(""), false},
		{"unknown is invalid", MatchType("unknown"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mt.IsValid())
		})
	}
}

func TestRouterMatch_HasAll(t *testing.T) {
	m := RouterMatch{}
	assert.False(t, m.HasAll())

	m.All = []RouterCondition{{Type: MatchTypeHost, Value: "example.com"}}
	assert.True(t, m.HasAll())
}

func TestRouterMatch_HasAny(t *testing.T) {
	m := RouterMatch{}
	assert.False(t, m.HasAny())

	m.Any = []RouterCondition{{Type: MatchTypeHost, Value: "example.com"}}
	assert.True(t, m.HasAny())
}

func TestRouterMatch_HasNot(t *testing.T) {
	m := RouterMatch{}
	assert.False(t, m.HasNot())

	m.Not = &RouterCondition{Type: MatchTypeHost, Value: "blocked.com"}
	assert.True(t, m.HasNot())
}

func TestRouter_Validate(t *testing.T) {
	for _, tt := range []struct {
		name        string
		router      Router
		expectedErr string
	}{
		{
			name: "valid shorthand rule",
			router: Router{
				Rules: []*RouterRule{
					{Id: "r1", Match: RouterMatch{Type: MatchTypeHost, Value: "example.com"}, Target: "pool-a"},
				},
			},
			expectedErr: "",
		},
		{
			name: "valid catch_all without value",
			router: Router{
				Rules: []*RouterRule{
					{Id: "r1", Match: RouterMatch{Type: MatchTypeCatchAll}, Target: "pool-default"},
				},
			},
			expectedErr: "",
		},
		{
			name: "valid all conditions",
			router: Router{
				Rules: []*RouterRule{
					{
						Id: "r1",
						Match: RouterMatch{
							All: []RouterCondition{
								{Type: MatchTypeHost, Value: "example.com"},
								{Type: MatchTypeMethod, Value: "GET"},
							},
						},
						Target: "pool-a",
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "valid any conditions",
			router: Router{
				Rules: []*RouterRule{
					{
						Id: "r1",
						Match: RouterMatch{
							Any: []RouterCondition{
								{Type: MatchTypeHost, Value: "a.com"},
								{Type: MatchTypeHost, Value: "b.com"},
							},
						},
						Target: "pool-a",
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "no match condition defined",
			router: Router{
				Rules: []*RouterRule{
					{Id: "r1", Match: RouterMatch{}, Target: "pool-a"},
				},
			},
			expectedErr: `rule "r1": match must define type, all, or any`,
		},
		{
			name: "shorthand type without value",
			router: Router{
				Rules: []*RouterRule{
					{Id: "r1", Match: RouterMatch{Type: MatchTypeHost}, Target: "pool-a"},
				},
			},
			expectedErr: `rule "r1": match type "host" requires a value`,
		},
		{
			name: "invalid condition type in all",
			router: Router{
				Rules: []*RouterRule{
					{
						Id: "r1",
						Match: RouterMatch{
							All: []RouterCondition{
								{Type: MatchType("bogus"), Value: "x"},
							},
						},
						Target: "pool-a",
					},
				},
			},
			expectedErr: `rule "r1": invalid condition type "bogus" in all`,
		},
		{
			name: "invalid condition type in any",
			router: Router{
				Rules: []*RouterRule{
					{
						Id: "r1",
						Match: RouterMatch{
							Any: []RouterCondition{
								{Type: MatchType("nope"), Value: "x"},
							},
						},
						Target: "pool-a",
					},
				},
			},
			expectedErr: `rule "r1": invalid condition type "nope" in any`,
		},
		{
			name: "invalid condition type in not",
			router: Router{
				Rules: []*RouterRule{
					{
						Id: "r1",
						Match: RouterMatch{
							Type:  MatchTypeHost,
							Value: "example.com",
							Not:   &RouterCondition{Type: MatchType("bad")},
						},
						Target: "pool-a",
					},
				},
			},
			expectedErr: `rule "r1": invalid condition type "bad" in not`,
		},
		{
			name: "missing target",
			router: Router{
				Rules: []*RouterRule{
					{Id: "r1", Match: RouterMatch{Type: MatchTypeCatchAll}, Target: ""},
				},
			},
			expectedErr: `rule "r1": target is required`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.router.Validate()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRouter_Resolve(t *testing.T) {
	router := Router{
		Rules: []*RouterRule{
			{
				Id:     "youtube",
				Match:  RouterMatch{Type: MatchTypeHost, Value: "youtube.com"},
				Target: "pool-video",
			},
			{
				Id:     "de-suffix",
				Match:  RouterMatch{Type: MatchTypeHostSuffix, Value: "de"},
				Target: "pool-de",
			},
			{
				Id:     "fallback",
				Match:  RouterMatch{Type: MatchTypeCatchAll},
				Target: "pool-default",
			},
		},
	}

	for _, tt := range []struct {
		name           string
		host           string
		expectedTarget string
		expectedFound  bool
	}{
		{
			name:           "matches host exactly",
			host:           "youtube.com",
			expectedTarget: "pool-video",
			expectedFound:  true,
		},
		{
			name:           "matches host suffix",
			host:           "example.de",
			expectedTarget: "pool-de",
			expectedFound:  true,
		},
		{
			name:           "falls through to catch_all",
			host:           "unknown.org",
			expectedTarget: "pool-default",
			expectedFound:  true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://"+tt.host+"/", nil)
			target, found := router.Resolve(req)
			assert.Equal(t, tt.expectedTarget, target)
			assert.Equal(t, tt.expectedFound, found)
		})
	}
}

func TestRouter_Resolve_NoMatch(t *testing.T) {
	router := Router{
		Rules: []*RouterRule{
			{
				Id:     "only-youtube",
				Match:  RouterMatch{Type: MatchTypeHost, Value: "youtube.com"},
				Target: "pool-video",
			},
		},
	}

	req, _ := http.NewRequest("GET", "http://google.com/", nil)
	target, found := router.Resolve(req)
	assert.Empty(t, target)
	assert.False(t, found)
}

func TestRouterMatch_Evaluate(t *testing.T) {
	for _, tt := range []struct {
		name     string
		match    RouterMatch
		req      *http.Request
		expected bool
	}{
		{
			name:     "catch_all matches everything",
			match:    RouterMatch{Type: MatchTypeCatchAll},
			req:      newReq("GET", "http://anything.com/path", nil),
			expected: true,
		},
		{
			name:     "host exact match",
			match:    RouterMatch{Type: MatchTypeHost, Value: "example.com"},
			req:      newReq("GET", "http://example.com/", nil),
			expected: true,
		},
		{
			name:     "host no match",
			match:    RouterMatch{Type: MatchTypeHost, Value: "example.com"},
			req:      newReq("GET", "http://other.com/", nil),
			expected: false,
		},
		{
			name:     "host_suffix matches domain",
			match:    RouterMatch{Type: MatchTypeHostSuffix, Value: "de"},
			req:      newReq("GET", "http://shop.example.de/", nil),
			expected: true,
		},
		{
			name:     "host_suffix matches exact domain",
			match:    RouterMatch{Type: MatchTypeHostSuffix, Value: "example.de"},
			req:      newReq("GET", "http://example.de/", nil),
			expected: true,
		},
		{
			name:     "host_suffix no match",
			match:    RouterMatch{Type: MatchTypeHostSuffix, Value: "de"},
			req:      newReq("GET", "http://example.com/", nil),
			expected: false,
		},
		{
			name:     "host_suffix with leading dot",
			match:    RouterMatch{Type: MatchTypeHostSuffix, Value: ".de"},
			req:      newReq("GET", "http://example.de/", nil),
			expected: true,
		},
		{
			name:     "path_prefix match",
			match:    RouterMatch{Type: MatchTypePathPrefix, Value: "/api/"},
			req:      newReq("GET", "http://x.com/api/v1/users", nil),
			expected: true,
		},
		{
			name:     "path_prefix no match",
			match:    RouterMatch{Type: MatchTypePathPrefix, Value: "/api/"},
			req:      newReq("GET", "http://x.com/web/home", nil),
			expected: false,
		},
		{
			name:     "path_regex match",
			match:    RouterMatch{Type: MatchTypePathRegex, Value: `^/v\d+/`},
			req:      newReq("GET", "http://x.com/v2/resource", nil),
			expected: true,
		},
		{
			name:     "path_regex no match",
			match:    RouterMatch{Type: MatchTypePathRegex, Value: `^/v\d+/`},
			req:      newReq("GET", "http://x.com/api/resource", nil),
			expected: false,
		},
		{
			name:     "method match",
			match:    RouterMatch{Type: MatchTypeMethod, Value: "POST"},
			req:      newReq("POST", "http://x.com/", nil),
			expected: true,
		},
		{
			name:     "method match case insensitive",
			match:    RouterMatch{Type: MatchTypeMethod, Value: "post"},
			req:      newReq("POST", "http://x.com/", nil),
			expected: true,
		},
		{
			name:     "method no match",
			match:    RouterMatch{Type: MatchTypeMethod, Value: "DELETE"},
			req:      newReq("GET", "http://x.com/", nil),
			expected: false,
		},
		{
			name:  "header match",
			match: RouterMatch{Type: MatchTypeHeader, Value: "X-Region:de"},
			req: newReq("GET", "http://x.com/", map[string]string{
				"X-Region": "de",
			}),
			expected: true,
		},
		{
			name:  "header no match",
			match: RouterMatch{Type: MatchTypeHeader, Value: "X-Region:de"},
			req: newReq("GET", "http://x.com/", map[string]string{
				"X-Region": "us",
			}),
			expected: false,
		},
		{
			name:     "header missing",
			match:    RouterMatch{Type: MatchTypeHeader, Value: "X-Region:de"},
			req:      newReq("GET", "http://x.com/", nil),
			expected: false,
		},
		{
			name:     "header invalid format",
			match:    RouterMatch{Type: MatchTypeHeader, Value: "NoColon"},
			req:      newReq("GET", "http://x.com/", nil),
			expected: false,
		},
		{
			name: "all conditions AND logic - all match",
			match: RouterMatch{
				All: []RouterCondition{
					{Type: MatchTypeHost, Value: "example.com"},
					{Type: MatchTypeMethod, Value: "GET"},
				},
			},
			req:      newReq("GET", "http://example.com/", nil),
			expected: true,
		},
		{
			name: "all conditions AND logic - one fails",
			match: RouterMatch{
				All: []RouterCondition{
					{Type: MatchTypeHost, Value: "example.com"},
					{Type: MatchTypeMethod, Value: "POST"},
				},
			},
			req:      newReq("GET", "http://example.com/", nil),
			expected: false,
		},
		{
			name: "any conditions OR logic - one matches",
			match: RouterMatch{
				Any: []RouterCondition{
					{Type: MatchTypeHost, Value: "a.com"},
					{Type: MatchTypeHost, Value: "b.com"},
				},
			},
			req:      newReq("GET", "http://b.com/", nil),
			expected: true,
		},
		{
			name: "any conditions OR logic - none matches",
			match: RouterMatch{
				Any: []RouterCondition{
					{Type: MatchTypeHost, Value: "a.com"},
					{Type: MatchTypeHost, Value: "b.com"},
				},
			},
			req:      newReq("GET", "http://c.com/", nil),
			expected: false,
		},
		{
			name: "not condition blocks match",
			match: RouterMatch{
				Type: MatchTypeCatchAll,
				Not:  &RouterCondition{Type: MatchTypeHost, Value: "blocked.com"},
			},
			req:      newReq("GET", "http://blocked.com/", nil),
			expected: false,
		},
		{
			name: "not condition allows match",
			match: RouterMatch{
				Type: MatchTypeCatchAll,
				Not:  &RouterCondition{Type: MatchTypeHost, Value: "blocked.com"},
			},
			req:      newReq("GET", "http://allowed.com/", nil),
			expected: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.match.Evaluate(tt.req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRouter_Resolve_FirstMatchWins(t *testing.T) {
	router := Router{
		Rules: []*RouterRule{
			{Id: "r1", Match: RouterMatch{Type: MatchTypeHost, Value: "example.com"}, Target: "first"},
			{Id: "r2", Match: RouterMatch{Type: MatchTypeHost, Value: "example.com"}, Target: "second"},
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	target, found := router.Resolve(req)

	require.True(t, found)
	assert.Equal(t, "first", target)
}

func TestRouter_Resolve_EmptyRules(t *testing.T) {
	router := Router{Rules: []*RouterRule{}}

	req, _ := http.NewRequest("GET", "http://example.com/", nil)
	target, found := router.Resolve(req)

	assert.Empty(t, target)
	assert.False(t, found)
}

// newReq is a test helper that creates an *http.Request with optional headers.
func newReq(method, url string, headers map[string]string) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

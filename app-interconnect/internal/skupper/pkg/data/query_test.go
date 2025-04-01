package data

import (
	"reflect"
	"testing"
)

func TestConsoleDataMerge(t *testing.T) {
	data := ConsoleData{}
	one := SiteQueryData{
		Site: Site{
			SiteName: "one",
			SiteId:   "1",
			Version:  "abc",
			Connected: []string{
				"two",
			},
			Namespace: "default",
			Url:       "http:some.cluster.com",
			Edge:      false,
		},
		TcpServices: []TcpService{
			{
				Service: Service{
					Address:  "a",
					Protocol: "tcp",
					Targets: []ServiceTarget{
						{
							Target: "c2",
							Name:   "3.3.3.3",
							SiteId: "one",
						},
						{
							Target: "c3",
							Name:   "4.4.4.4",
							SiteId: "one",
						},
					},
				},
				ConnectionsIngress: []TcpServiceEndpoints{
					{
						SiteId: "one",
						Connections: map[string]TcpConnectionStats{
							"a1": {
								Id:        "a1",
								StartTime: 60,
								LastOut:   4,
								LastIn:    5,
								BytesIn:   10,
								BytesOut:  20,
								Client:    "1.1.1.1",
							},
							"a2": {
								Id:        "a2",
								StartTime: 90,
								LastOut:   9,
								LastIn:    10,
								BytesIn:   40,
								BytesOut:  80,
								Client:    "2.2.2.2",
							},
						},
					},
				},
				ConnectionsEgress: []TcpServiceEndpoints{
					{
						SiteId: "one",
						Connections: map[string]TcpConnectionStats{
							"a1": {
								Id:        "a1",
								StartTime: 60,
								LastOut:   4,
								LastIn:    5,
								BytesIn:   20,
								BytesOut:  10,
								Server:    "3.3.3.3",
							},
							"a2": {
								Id:        "a2",
								StartTime: 90,
								LastOut:   10,
								LastIn:    9,
								BytesIn:   80,
								BytesOut:  40,
								Server:    "4.4.4.4",
							},
						},
					},
				},
			},
			{
				Service: Service{
					Address:  "b",
					Protocol: "tcp",
					Targets: []ServiceTarget{
						{
							Target: "c1",
							Name:   "1.1.1.1",
							SiteId: "one",
						},
					},
				},
				ConnectionsIngress: []TcpServiceEndpoints{
					{
						SiteId: "one",
						Connections: map[string]TcpConnectionStats{
							"b1": {
								Id:        "b1",
								StartTime: 120,
								LastOut:   1,
								LastIn:    1,
								BytesIn:   15,
								BytesOut:  15,
								Client:    "5.5.5.5",
							},
						},
					},
				},
				ConnectionsEgress: []TcpServiceEndpoints{
					{
						SiteId: "one",
						Connections: map[string]TcpConnectionStats{
							"b1": {
								Id:        "b1",
								StartTime: 120,
								LastOut:   1,
								LastIn:    1,
								BytesIn:   15,
								BytesOut:  15,
								Server:    "1.1.1.1",
							},
						},
					},
				},
			},
			{
				Service: Service{
					Address:  "c",
					Protocol: "tcp",
					Targets: []ServiceTarget{
						{
							Target: "c4",
							Name:   "6.6.6.6",
							SiteId: "one",
						},
					},
				},
			},
		},
		HttpServices: []HttpService{
			{
				Service: Service{
					Address:  "foo",
					Protocol: "http",
					Targets: []ServiceTarget{
						{
							Target: "c1",
							Name:   "3.3.3.3",
							SiteId: "one",
						},
						{
							Target: "c2",
							Name:   "myhost",
							SiteId: "one",
						},
					},
				},
				RequestsReceived: []HttpRequestsReceived{
					{
						SiteId: "one",
						ByClient: map[string]HttpRequestStats{
							"1.1.1.1": {
								Requests: 5,
								BytesIn:  100,
								BytesOut: 2500,
								Details: map[string]int{
									"GET:200":  3,
									"POST:404": 2,
								},
								LatencyMax: 10,
								ByHandlingSite: map[string]HttpRequestStats{
									"one": {
										Requests: 5,
										BytesIn:  100,
										BytesOut: 2500,
										Details: map[string]int{
											"GET:200":  3,
											"POST:404": 2,
										},
										LatencyMax: 10,
									},
								},
							},
						},
					},
				},
				RequestsHandled: []HttpRequestsHandled{
					{
						SiteId: "one",
						ByServer: map[string]HttpRequestStats{
							"myhost": {
								Requests: 3,
								BytesIn:  70,
								BytesOut: 2300,
								Details: map[string]int{
									"GET:200":  2,
									"POST:404": 1,
								},
								LatencyMax: 10,
							},
							"3.3.3.3": {
								Requests: 2,
								BytesIn:  30,
								BytesOut: 200,
								Details: map[string]int{
									"GET:200":  1,
									"POST:404": 1,
								},
								LatencyMax: 15,
							},
						},
						ByOriginatingSite: map[string]HttpRequestStats{
							"one": {
								Requests: 5,
								BytesIn:  100,
								BytesOut: 2500,
								Details: map[string]int{
									"GET:200":  3,
									"POST:404": 2,
								},
								LatencyMax: 15,
							},
						},
					},
				},
			},
			{
				Service: Service{
					Address:  "bar",
					Protocol: "http2",
					Targets: []ServiceTarget{
						{
							Target: "c3",
							Name:   "",
							SiteId: "one",
						},
					},
				},
				RequestsReceived: []HttpRequestsReceived{
					{
						SiteId: "one",
						ByClient: map[string]HttpRequestStats{
							"1.1.1.1": {
								Requests: 4,
								BytesIn:  400,
								BytesOut: 400,
								Details: map[string]int{
									"GET:200": 2,
									"PUT:201": 2,
								},
								LatencyMax: 3,
								ByHandlingSite: map[string]HttpRequestStats{
									"one": {
										Requests: 4,
										BytesIn:  400,
										BytesOut: 400,
										Details: map[string]int{
											"GET:200": 2,
											"PUT:201": 2,
										},
										LatencyMax: 3,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	two := SiteQueryData{
		Site: Site{
			SiteName:  "two",
			SiteId:    "2",
			Version:   "def",
			Connected: []string{},
			Namespace: "myspace",
			Url:       "http:some.other.cluster.com",
			Edge:      false,
		},
		TcpServices: []TcpService{
			{
				Service: Service{
					Address:  "a",
					Protocol: "tcp",
					Targets: []ServiceTarget{
						{
							Target: "c5",
							Name:   "7.7.7.7",
							SiteId: "two",
						},
					},
				},
				ConnectionsIngress: []TcpServiceEndpoints{
					{
						SiteId: "two",
						Connections: map[string]TcpConnectionStats{
							"a3": {
								Id:        "a3",
								StartTime: 60,
								LastOut:   4,
								LastIn:    5,
								BytesIn:   10,
								BytesOut:  20,
								Client:    "1.1.1.1",
							},
						},
					},
				},
				ConnectionsEgress: []TcpServiceEndpoints{
					{
						SiteId: "two",
						Connections: map[string]TcpConnectionStats{
							"a3": {
								Id:        "a3",
								StartTime: 60,
								LastOut:   4,
								LastIn:    5,
								BytesIn:   20,
								BytesOut:  10,
								Server:    "3.3.3.3",
							},
						},
					},
				},
			},
			{
				Service: Service{
					Address:  "c",
					Protocol: "tcp",
					Targets: []ServiceTarget{
						{
							Target: "c7",
							Name:   "6.6.6.6",
							SiteId: "two",
						},
					},
				},
				ConnectionsIngress: []TcpServiceEndpoints{
					{
						SiteId: "two",
						Connections: map[string]TcpConnectionStats{
							"c2": {
								Id:        "c2",
								StartTime: 60,
								LastOut:   4,
								LastIn:    5,
								BytesIn:   10,
								BytesOut:  20,
								Client:    "1.1.1.1",
							},
						},
					},
				},
				ConnectionsEgress: []TcpServiceEndpoints{
					{
						SiteId: "two",
						Connections: map[string]TcpConnectionStats{
							"c2": {
								Id:        "c2",
								StartTime: 60,
								LastOut:   4,
								LastIn:    5,
								BytesIn:   20,
								BytesOut:  10,
								Server:    "3.3.3.3",
							},
						},
					},
				},
			},
		},
		HttpServices: []HttpService{
			{
				Service: Service{
					Address:  "foo",
					Protocol: "http",
					Targets: []ServiceTarget{
						{
							Target: "c10",
							Name:   "3.3.3.3",
							SiteId: "two",
						},
					},
				},
			},
			{
				Service: Service{
					Address:  "bar",
					Protocol: "http2",
					Targets: []ServiceTarget{
						{
							Target: "c10",
							Name:   "3.3.3.3",
							SiteId: "two",
						},
					},
				},
			},
		},
	}

	sites := []SiteQueryData{
		one,
		two,
	}
	data.Merge(sites)
	expectedSites := map[string]Site{
		"one": {
			SiteName: "one",
			SiteId:   "1",
			Version:  "abc",
			Connected: []string{
				"two",
			},
			Namespace: "default",
			Url:       "http:some.cluster.com",
			Edge:      false,
		},
		"two": {
			SiteName:  "two",
			SiteId:    "2",
			Version:   "def",
			Connected: []string{},
			Namespace: "myspace",
			Url:       "http:some.other.cluster.com",
			Edge:      false,
		},
	}
	if len(data.Sites) != len(expectedSites) {
		t.Errorf("Expected %d sites, got %d", len(data.Sites), len(expectedSites))
	}
	for _, s := range data.Sites {
		e := expectedSites[s.SiteName]
		if !reflect.DeepEqual(e, s) {
			t.Errorf("Incorrect site for %s; expected %v, got %v", s.SiteName, e, s)
		}
	}
	expectedServices := map[string]Service{
		"a": {
			Address:  "a",
			Protocol: "tcp",
		},
		"b": {
			Address:  "b",
			Protocol: "tcp",
		},
		"c": {
			Address:  "c",
			Protocol: "tcp",
		},
		"foo": {
			Address:  "foo",
			Protocol: "http",
		},
		"bar": {
			Address:  "bar",
			Protocol: "http2",
		},
	}
	if len(data.Services) != len(expectedServices) {
		t.Errorf("Expected %d services, got %d", len(data.Services), len(expectedServices))
	}
	for _, s := range data.Services {
		var service Service
		matched := false
		if t, ok := s.(TcpService); ok {
			service = t.Service
			matched = true
		} else if h, ok := s.(HttpService); ok {
			service = h.Service
			matched = true
		}
		if !matched {
			t.Errorf("Unrecognised service item %v", s)
		} else {
			e := expectedServices[service.Address]
			if e.Address != service.Address {
				t.Errorf("Incorrect service address; expected %s, got %s", e.Address, service.Address)
			}
			if e.Protocol != service.Protocol {
				t.Errorf("Incorrect service protocol; expected %s, got %s", e.Protocol, service.Protocol)
			}
		}
	}
}

func TestAsLegacySiteInfo(t *testing.T) {
	one := Site{
		SiteName: "one",
		SiteId:   "1",
		Version:  "abc",
		Connected: []string{
			"two",
		},
		Namespace: "default",
		Url:       "http:some.cluster.com",
		Edge:      false,
	}
	legacy := one.AsLegacySiteInfo()
	expected := LegacySiteInfo{
		SiteId:    one.SiteId,
		SiteName:  one.SiteName,
		Version:   one.Version,
		Namespace: one.Namespace,
		Url:       one.Url,
	}
	if expected != *legacy {
		t.Errorf("AsLegacySiteInfo returned invalid result; expected %v, got %v", expected, legacy)
	}
}

func TestNullMapping(t *testing.T) {
	m := NewNullNameMapping()
	if "foo" != m.Lookup("foo") {
		t.Errorf("Null mapping doesn't work!")
	}
}

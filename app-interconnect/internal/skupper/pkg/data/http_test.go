package data

import (
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/qdr"
	"reflect"
	"testing"
)

type TestMapping struct {
	mapping map[string]string
}

func (t *TestMapping) Lookup(name string) string {
	if v, ok := t.mapping[name]; ok {
		return v
	} else {
		return name
	}
}

func GetTestMapping(mapping map[string]string) NameMapping {
	return &TestMapping{
		mapping: mapping,
	}
}

func TestGetHttpServices(t *testing.T) {
	siteId := "mysite"
	router1Stats := []qdr.HttpRequestInfo{
		{
			Name:       "ai",
			Host:       "1.1.1.1",
			Address:    "foo",
			Site:       siteId,
			Direction:  "in",
			Requests:   5,
			BytesIn:    100,
			BytesOut:   2500,
			MaxLatency: 10,
			Details: map[string]int{
				"GET:200":  3,
				"POST:404": 2,
			},
		},
		{
			Name:       "bi1",
			Host:       "1.1.1.1",
			Address:    "bar",
			Site:       siteId,
			Direction:  "in",
			Requests:   3,
			BytesIn:    300,
			BytesOut:   300,
			MaxLatency: 3,
			Details: map[string]int{
				"GET:200": 1,
				"PUT:201": 2,
			},
		},
		{
			Name:       "bi2",
			Host:       "1.1.1.1",
			Address:    "bar",
			Site:       siteId,
			Direction:  "in",
			Requests:   1,
			BytesIn:    100,
			BytesOut:   100,
			MaxLatency: 2,
			Details: map[string]int{
				"GET:200": 1,
			},
		},
	}
	router2Stats := []qdr.HttpRequestInfo{
		{
			Name:       "ao1",
			Host:       "2.2.2.2",
			Address:    "foo",
			Site:       siteId,
			Direction:  "out",
			Requests:   3,
			BytesIn:    70,
			BytesOut:   2300,
			MaxLatency: 10,
			Details: map[string]int{
				"GET:200":  2,
				"POST:404": 1,
			},
		},
		{
			Name:       "ao2",
			Host:       "3.3.3.3",
			Address:    "foo",
			Site:       siteId,
			Direction:  "out",
			Requests:   1,
			BytesIn:    20,
			BytesOut:   180,
			MaxLatency: 15,
			Details: map[string]int{
				"GET:200": 1,
			},
		},
		{
			Name:       "ao2",
			Host:       "3.3.3.3",
			Address:    "foo",
			Site:       siteId,
			Direction:  "out",
			Requests:   1,
			BytesIn:    10,
			BytesOut:   20,
			MaxLatency: 5,
			Details: map[string]int{
				"POST:404": 1,
			},
		},
	}
	routerStats := [][]qdr.HttpRequestInfo{
		router1Stats,
		router2Stats,
	}
	targets := []qdr.HttpEndpoint{
		{
			Name:    "c1",
			Host:    "3.3.3.3",
			Address: "foo",
			SiteId:  siteId,
		},
		{
			Name:    "c2",
			Host:    "2.2.2.2",
			Address: "foo",
			SiteId:  siteId,
		},
		{
			Name:    "c3",
			Host:    "4.4.4.4",
			Address: "bar",
			SiteId:  siteId,
		},
	}
	listeners := []qdr.HttpEndpoint{
		{
			Name:    "l1",
			Address: "foo",
			SiteId:  siteId,
		},
		{
			Name:            "l2",
			Address:         "bar",
			ProtocolVersion: qdr.HttpVersion2,
			SiteId:          siteId,
		},
	}
	mapping := GetTestMapping(map[string]string{
		"2.2.2.2": "myhost",
		"4.4.4.4": "",
	})

	services := GetHttpServices(siteId, routerStats, targets, listeners, mapping)
	if services == nil {
		t.Errorf("Got nil services list")
	}
	expected := map[string]HttpService{
		"foo": {
			Service: Service{
				Address:  "foo",
				Protocol: "http",
				Targets: []ServiceTarget{
					{
						Target: "c1",
						Name:   "3.3.3.3",
						SiteId: siteId,
					},
					{
						Target: "c2",
						Name:   "myhost",
						SiteId: siteId,
					},
				},
			},
			RequestsReceived: []HttpRequestsReceived{
				{
					SiteId: siteId,
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
								siteId: {
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
					SiteId: siteId,
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
						siteId: {
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
		"bar": {
			Service: Service{
				Address:  "bar",
				Protocol: "http2",
				Targets: []ServiceTarget{
					{
						Target: "c3",
						Name:   "",
						SiteId: siteId,
					},
				},
			},
			RequestsReceived: []HttpRequestsReceived{
				{
					SiteId: siteId,
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
								siteId: {
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
	}
	if len(services) != len(expected) {
		t.Errorf("Expected %d services, got %d", len(expected), len(services))
	}
	for _, s := range services {
		e := expected[s.Address]
		if !reflect.DeepEqual(e.Service, s.Service) {
			t.Errorf("Incorrect service definition for %s; expected %v, got %v", e.Service.Address, e.Service, s.Service)
		}
		if !reflect.DeepEqual(e.RequestsReceived, s.RequestsReceived) {
			t.Errorf("Incorrect requests-received for %s; expected %v, got %v", e.Service.Address, e.RequestsReceived, s.RequestsReceived)
		}
		if !reflect.DeepEqual(e.RequestsHandled, s.RequestsHandled) {
			t.Errorf("Incorrect requests-handled for %s; expected %v, got %v", e.Service.Address, e.RequestsHandled, s.RequestsHandled)
		}
	}
}

func TestTargetDefinesService(t *testing.T) {
	siteId := "whatever"
	routerStats := [][]qdr.HttpRequestInfo{}
	targets := []qdr.HttpEndpoint{
		{
			Name:    "c1",
			Host:    "3.3.3.3",
			Address: "foo",
			SiteId:  siteId,
		},
	}
	listeners := []qdr.HttpEndpoint{}
	mapping := NewNullNameMapping()
	services := GetHttpServices(siteId, routerStats, targets, listeners, mapping)
	if services == nil {
		t.Errorf("Got nil services list")
	}
	expected := map[string]HttpService{
		"foo": {
			Service: Service{
				Address:  "foo",
				Protocol: "http",
				Targets: []ServiceTarget{
					{
						Target: "c1",
						Name:   "3.3.3.3",
						SiteId: siteId,
					},
				},
			},
		},
	}
	for _, s := range services {
		e := expected[s.Address]
		if !reflect.DeepEqual(e.Service, s.Service) {
			t.Errorf("Incorrect service definition for %s; expected %v, got %v", e.Service.Address, e.Service, s.Service)
		}
	}
}

package domain

import (
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/qdr"
)

// RouterEntityManager manipulates runtime entities
type RouterEntityManager interface {
	CreateSslProfile(sslProfile qdr.SslProfile) error
	DeleteSslProfile(name string) error
	CreateConnector(connector qdr.Connector) error
	DeleteConnector(name string) error
	QueryAllRouters() ([]qdr.Router, error)
	QueryRouterNodes() ([]qdr.RouterNode, error)
	QueryEdgeRouters() ([]qdr.Router, error)
	QueryConnections(routerId string, edge bool) ([]qdr.Connection, error)
	CreateTcpConnector(tcpConnector qdr.TcpEndpoint) error
	DeleteTcpConnector(name string) error
	CreateHttpConnector(httpConnector qdr.HttpEndpoint) error
	DeleteHttpConnector(name string) error
}

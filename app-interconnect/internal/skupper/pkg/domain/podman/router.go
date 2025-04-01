package podman

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/api/types"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/client/podman"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/qdr"
)

type RouterEntityManager struct {
	cli       *podman.PodmanRestClient
	container string
}

func NewRouterEntityManagerPodman(cli *podman.PodmanRestClient) *RouterEntityManager {
	return NewRouterEntityManagerPodmanFor(cli, types.TransportDeploymentName)
}

func NewRouterEntityManagerPodmanFor(cli *podman.PodmanRestClient, container string) *RouterEntityManager {
	return &RouterEntityManager{
		cli:       cli,
		container: container,
	}
}

func (r *RouterEntityManager) exec(cmd []string) (string, error) {
	return r.cli.ContainerExec(r.container, cmd)
}

func (r *RouterEntityManager) CreateSslProfile(sslProfile qdr.SslProfile) error {
	cmd := qdr.SkmanageCreateCommand("sslProfile", sslProfile.Name, sslProfile)
	if _, err := r.exec(cmd); err != nil {
		return fmt.Errorf("error creating sslProfile %s - %w", sslProfile.Name, err)
	}
	return nil
}

func (r *RouterEntityManager) DeleteSslProfile(name string) error {
	cmd := qdr.SkmanageDeleteCommand("sslProfile", name)
	if _, err := r.exec(cmd); err != nil {
		return fmt.Errorf("error deleting sslProfile %s - %w", name, err)
	}
	return nil
}

func (r *RouterEntityManager) CreateConnector(connector qdr.Connector) error {
	cmd := qdr.SkmanageCreateCommand("connector", connector.Name, connector)
	if _, err := r.exec(cmd); err != nil {
		return fmt.Errorf("error creating connector %s - %w", connector.Name, err)
	}
	return nil
}

func (r *RouterEntityManager) DeleteConnector(name string) error {
	cmd := qdr.SkmanageDeleteCommand("connector", name)
	if _, err := r.exec(cmd); err != nil {
		return fmt.Errorf("error deleting sslProfile %s - %w", name, err)
	}
	return nil
}

func (r *RouterEntityManager) QueryConnections(routerId string, edge bool) ([]qdr.Connection, error) {
	cmd := qdr.SkmanageQueryCommand("connection", routerId, edge, "")
	var data string
	var err error
	if data, err = r.exec(cmd); err != nil {
		return nil, fmt.Errorf("error querying connections - %w", err)
	}
	var connections []qdr.Connection
	err = json.Unmarshal([]byte(data), &connections)
	if err != nil {
		if r.isInvalidResponseFromStaleRouter(data) {
			fmt.Printf("Warning: unable to retrieve connections from router %q, as it is no longer available", routerId)
			fmt.Println()
		} else {
			return nil, fmt.Errorf("error retrieving connections - %w - output: %q", err, data)
		}
	}
	return connections, nil
}

func (r *RouterEntityManager) QueryAllRouters() ([]qdr.Router, error) {
	var routersToQuery []qdr.Router
	var routersTmp []qdr.Router
	routerNodes, err := r.QueryRouterNodes()
	if err != nil {
		return nil, err
	}
	edgeRouters, err := r.QueryEdgeRouters()
	if err != nil {
		return nil, err
	}
	for _, r := range routerNodes {
		routersToQuery = append(routersToQuery, *r.AsRouter())
	}
	for _, r := range edgeRouters {
		routersToQuery = append(routersToQuery, r)
	}
	var nodeIds []string
	for _, router := range routersToQuery {
		// querying io.skupper.router.router to retrieve version for all routers found
		routerToQuery := router.Id
		cmd := qdr.SkmanageQueryCommand("io.skupper.router.router", routerToQuery, router.Edge, "")
		rJson, err := r.cli.ContainerExec(types.TransportDeploymentName, cmd)
		if err != nil {
			return nil, fmt.Errorf("error querying router info from %s - %w", routerToQuery, err)
		}
		var records []qdr.Record
		err = json.Unmarshal([]byte(rJson), &records)
		if err != nil {
			if r.isInvalidResponseFromStaleRouter(rJson) {
				continue
			}
			return nil, fmt.Errorf("error decoding router info from %s - %w - %s", routerToQuery, err, rJson)
		}
		router.Site = qdr.GetSiteMetadata(records[0].AsString("metadata"))

		// retrieving connections
		conns, err := r.QueryConnections(routerToQuery, router.Edge)
		if err != nil {
			return nil, fmt.Errorf("error querying router connections from %s - %w", routerToQuery, err)
		}
		for _, conn := range conns {
			if conn.Role == types.InterRouterRole && conn.Dir == qdr.DirectionOut {
				router.ConnectedTo = append(router.ConnectedTo, conn.Container)
			}
		}
		if !router.Edge {
			nodeIds = append(nodeIds, router.Site.Id)
		} else {
			svcRouter := false
			for _, nodeId := range nodeIds {
				// Podman svc router
				if strings.HasPrefix(router.Site.Id, nodeId+"-") {
					svcRouter = true
					break
				}
				// Headless svc (kube)
				if router.Site.Id == nodeId {
					svcRouter = true
					break
				}
			}
			if svcRouter {
				continue
			}
		}
		routersTmp = append(routersTmp, router)
	}
	return routersTmp, nil
}

// isInvalidResponseFromStaleRouter returns true if response is not a valid JSON
// message returned by the management API, because the given router is stale as it
// is still showing up as an existing router.node entity, but the connection is
// no longer active.
func (r *RouterEntityManager) isInvalidResponseFromStaleRouter(jsonOutput string) bool {
	return strings.Contains(jsonOutput, "SendException: RELEASED") || strings.Contains(jsonOutput, "Timeout: ")
}

func (r *RouterEntityManager) QueryRouterNodes() ([]qdr.RouterNode, error) {
	var routerNodes []qdr.RouterNode
	// Retrieving all connections
	conns, err := r.QueryConnections("", false)
	if err != nil {
		return nil, fmt.Errorf("error retrieving router connections - %w", err)
	}
	// Retrieving Router nodes
	var routerId string
	var edge bool
	for _, conn := range conns {
		if conn.Role == types.ConnectorRoleEdge && conn.Dir == qdr.DirectionOut {
			routerId = conn.Container
			edge = true
			break
		}
	}
	cmd := qdr.SkmanageQueryCommand("io.skupper.router.router.node", routerId, edge, "")
	routerNodesJson, err := r.cli.ContainerExec(types.TransportDeploymentName, cmd)
	if err != nil {
		return nil, fmt.Errorf("error querying router nodes - %w", err)
	}
	err = json.Unmarshal([]byte(routerNodesJson), &routerNodes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse router nodes - %w", err)
	}
	return routerNodes, nil
}

func (r *RouterEntityManager) QueryEdgeRouters() ([]qdr.Router, error) {
	var routers []qdr.Router
	routerNodes, err := r.QueryRouterNodes()
	if err != nil {
		return nil, fmt.Errorf("error querying router nodes - %w", err)
	}
	for _, routerNode := range routerNodes {
		conns, err := r.QueryConnections(routerNode.Id, false)
		if err != nil {
			return nil, fmt.Errorf("error querying connections from router %s - %w", routerNode.Id, err)
		}
		for _, conn := range conns {
			if conn.Role == types.EdgeRole && conn.Dir == qdr.DirectionIn {
				routers = append(routers, qdr.Router{
					Id:          conn.Container,
					Address:     qdr.GetRouterAddress(conn.Container, true),
					Edge:        true,
					ConnectedTo: []string{routerNode.Id},
				})
			}
		}
	}
	return routers, nil
}

func (r *RouterEntityManager) CreateTcpConnector(tcpConnector qdr.TcpEndpoint) error {
	cmd := qdr.SkmanageCreateCommand("tcpConnector", tcpConnector.Name, tcpConnector)
	if _, err := r.exec(cmd); err != nil {
		return fmt.Errorf("error creating tcpConnector %s - %w", tcpConnector.Name, err)
	}
	return nil
}

func (r *RouterEntityManager) DeleteTcpConnector(name string) error {
	cmd := qdr.SkmanageDeleteCommand("tcpConnector", name)
	if _, err := r.exec(cmd); err != nil {
		return fmt.Errorf("error deleting tcpConnector %s - %w", name, err)
	}
	return nil
}

func (r *RouterEntityManager) CreateHttpConnector(httpConnector qdr.HttpEndpoint) error {
	cmd := qdr.SkmanageCreateCommand("httpConnector", httpConnector.Name, httpConnector)
	if _, err := r.exec(cmd); err != nil {
		return fmt.Errorf("error creating httpConnector %s - %w", httpConnector.Name, err)
	}
	return nil
}

func (r *RouterEntityManager) DeleteHttpConnector(name string) error {
	cmd := qdr.SkmanageDeleteCommand("httpConnector", name)
	if _, err := r.exec(cmd); err != nil {
		return fmt.Errorf("error deleting httpConnector %s - %w", name, err)
	}
	return nil
}

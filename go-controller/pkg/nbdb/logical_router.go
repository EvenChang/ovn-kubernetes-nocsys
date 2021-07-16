// Code generated by "libovsdb.modelgen"
// DO NOT EDIT.

package nbdb

// LogicalRouter defines an object in Logical_Router table
type LogicalRouter struct {
	UUID         string            `ovsdb:"_uuid"`
	Enabled      []bool            `ovsdb:"enabled"`
	ExternalIDs  map[string]string `ovsdb:"external_ids"`
	LoadBalancer []string          `ovsdb:"load_balancer"`
	Name         string            `ovsdb:"name"`
	Nat          []string          `ovsdb:"nat"`
	Options      map[string]string `ovsdb:"options"`
	Policies     []string          `ovsdb:"policies"`
	Ports        []string          `ovsdb:"ports"`
	StaticRoutes []string          `ovsdb:"static_routes"`
}

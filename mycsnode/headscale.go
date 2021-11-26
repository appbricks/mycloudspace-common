package mycsnode

type CreateMeshAuthKeyReq struct {
	ExpiresIn int64 `json:"expiresIn,omitempty"`
}
type CreateMeshAuthKeyResp struct {
	AuthKey      string   `json:"authKey,omitempty"`
	DNS          []string `json:"dns,omitempty"`
	SpaceNode    TSNode   `json:"space_node,omitempty"`
	DeviceNodes  []TSNode `json:"device_nodes,omitempty"`
}
type TSNode struct {
	Name   string   `json:"name,omitempty"`
	IP     string   `json:"ip,omitempty"`
	Routes []string `json:"routes,omitempty"`
}

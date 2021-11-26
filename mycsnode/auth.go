package mycsnode

type AuthRequest struct {
	DeviceIDKey string `json:"deviceIDKey"`
	AuthReqKey  string `json:"authReqKey"`
}
type AuthReqKey struct {
	UserID              string `json:"userID"`
	DeviceECDHPublicKey string `json:"deviceECDHPublicKey"`
	Nonce               int64  `json:"nonce"`
}
type AuthResponse struct {
	AuthRespKey string `json:"authRespKey"`
	AuthIDKey   string `json:"authIDKey"`
}
type AuthRespKey struct {
	NodeECDHPublicKey string `json:"nodeECDHPublicKey"`
	Nonce             int64  `json:"nonce"`
	TimeoutAt         int64  `json:"timeoutAt"`
	DeviceName        string `json:"deviceName"`
}

package types

type UpstreamRow struct {
	UpstreamId  int64
	UpstreamUrl string
	Online      int64
	Primary     int64
	Shadow      int64
}

type ShadowEndpointRow struct {
	Id       int64
	Endpoint string
}

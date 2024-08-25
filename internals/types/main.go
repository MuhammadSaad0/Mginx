package types

type UpstreamRow struct {
	UpstreamId  int64
	UpstreamUrl string
	Online      int64
	Primary     int64
	Shadow      int64
}

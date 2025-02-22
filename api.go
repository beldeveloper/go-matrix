package gomatrix

type apiLoginReq struct {
	Type     string `json:"type"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type apiSendMsgReq struct {
	RoomID        string `json:"-"`
	Type          string `json:"msgtype"`
	Body          string `json:"body,omitempty"`
	Format        string `json:"format,omitempty"`
	FormattedBody string `json:"formatted_body,omitempty"`
	Filename      string `json:"filename,omitempty"`
	URL           string `json:"url,omitempty"`
}

type apiUploadResp struct {
	URI string `json:"content_uri"`
}

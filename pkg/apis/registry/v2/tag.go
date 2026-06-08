package v2

type TagList struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func (TagList) ContentType() string {
	return "application/json"
}

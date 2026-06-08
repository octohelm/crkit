package v2

type TagList struct {
	// Name 仓库名称
	Name string `json:"name"`
	// Tags 标签列表
	Tags []string `json:"tags"`
}

func (TagList) ContentType() string {
	return "application/json"
}

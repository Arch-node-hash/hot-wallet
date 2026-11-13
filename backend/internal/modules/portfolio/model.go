package portfolio

type Stat struct {
	Title    string `json:"title"`
	Value    string `json:"value"`
	Subtitle string `json:"subtitle,omitempty"`
	Increase string `json:"increase,omitempty"`
	Change   string `json:"change,omitempty"`
	Trend    string `json:"trend,omitempty"`
}

type MonthlyPoint struct {
	Month    string `json:"month"`
	Tasks    int    `json:"tasks"`
	Projects int    `json:"projects"`
}

type DistributionItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Color string `json:"color"`
}

type Analytics struct {
	Stats        []Stat             `json:"stats"`
	Monthly      []MonthlyPoint     `json:"monthly"`
	Distribution []DistributionItem `json:"distribution"`
}

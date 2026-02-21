package store

type EntityInput struct {
	Name       string
	EntityType string
	Layer      string
	SourceFile string
	SourceHash string
	Properties map[string]any
	Tags       []string
	Body       string
}

type Entity struct {
	Name       string
	EntityType string
	Layer      string
	SourceFile string
	SourceHash string
	Tags       []string
	Properties map[string]any
	Body       string
}

type EntitySummary struct {
	Name       string
	EntityType string
	Layer      string
	Tags       []string
}

type EntityRef struct {
	Name       string
	EntityType string
	Layer      string
}

type Relationship struct {
	From      EntityRef
	To        EntityRef
	Type      string
	Direction string
	Depth     int
}

type SearchResult struct {
	Name       string
	EntityType string
	Layer      string
	Tags       []string
	Score      float64
	Snippet    string
}

type Consequence struct {
	Entity   string `json:"entity"`
	Property string `json:"property"`
	Value    any    `json:"value,omitempty"`
	Add      any    `json:"add,omitempty"`
}

type Event struct {
	Name         string
	Layer        string
	Session      int
	DateInWorld  string
	Participants []string
	Location     []string
	Consequences []Consequence
}

type CurrentState struct {
	BaseProperties    map[string]any
	Events            []Event
	CurrentProperties map[string]any
}

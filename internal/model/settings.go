package model

type Settings struct {
	Workspace, Token, Timezone string
	Projects                   map[int64]ProjectSettings
}

type ProjectSettings struct {
	DisplayName  string
	Hidden       bool
	DisplayOrder *float64
}

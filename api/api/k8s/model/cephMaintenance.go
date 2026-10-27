package model

// CephField describes a validated parameter and its form control.
type CephField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
	Hint     string   `json:"hint,omitempty"`
}

type CephOperation struct {
	ID          string      `json:"id"`
	Group       string      `json:"group"`
	Label       string      `json:"label"`
	Description string      `json:"description"`
	Dangerous   bool        `json:"dangerous"`
	Schedulable bool        `json:"schedulable"`
	Fields      []CephField `json:"fields"`
}

type CephTaskRequest struct {
	Operation      string            `json:"operation" binding:"required"`
	Params         map[string]string `json:"params"`
	Confirmation   string            `json:"confirmation"`
	TimeoutSeconds int64             `json:"timeoutSeconds"`
}

type CephScheduleRequest struct {
	Name     string          `json:"name" binding:"required"`
	Schedule string          `json:"schedule" binding:"required"`
	TimeZone string          `json:"timeZone"`
	Task     CephTaskRequest `json:"task" binding:"required"`
}

type CephRepositoryRequest struct {
	Name string `json:"name" binding:"required"`
	PVC  string `json:"pvc" binding:"required"`
}

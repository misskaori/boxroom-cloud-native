package tree

import "encoding/json"

type Status interface {
	SetStatus(status string)
	AddFailedObjects(name string, err error)
	CovertStructToJson() ([]byte, error)
	CovertJsonToStruct(jsonDefinition []byte) error
}

type MissionStatus struct {
	MissionKind   string
	Status        string
	FailedObjects map[string]error
}

func (m *MissionStatus) CovertStructToJson() ([]byte, error) {
	return json.Marshal(m)
}

func (m *MissionStatus) CovertJsonToStruct(jsonDefinition []byte) error {
	return json.Unmarshal(jsonDefinition, m)
}

func (m *MissionStatus) SetStatus(status string) {
	m.Status = status
}

func (m *MissionStatus) AddFailedObjects(name string, err error) {
	m.FailedObjects[name] = err
}

package model

import "time"

// VacuumReport is a serialized, ready to re-replay linting report. It can be used on its own, or it
// can be used as a replay model to re-render the report again. Time is now available to vacuum.
type VacuumReport struct {
	Generated time.Time      `json:"generated" yaml:"generated"`
	SpecInfo  *SpecInfo      `json:"specInfo" yaml:"specInfo"`
	ResultSet *RuleResultSet `json:"resultSet" yaml:"resultSet"`
}

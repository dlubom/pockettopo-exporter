package exportsvx

import (
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
)

type Identity struct {
	Raw     uint32 `json:"raw"`
	Native  int32  `json:"native"`
	Display string `json:"display"`
	SVX     string `json:"svx_name"`
}
type Measurement struct {
	Index            int         `json:"index"`
	Span             source.Span `json:"source_span"`
	Line             int         `json:"svx_line"`
	Head             int         `json:"group_head"`
	Reversed         bool        `json:"reversed_member"`
	From             Identity    `json:"from"`
	To               Identity    `json:"to"`
	Distance         int32       `json:"distance_mm"`
	Azimuth          int16       `json:"azimuth_raw"`
	Inclination      int16       `json:"inclination_raw"`
	Flags            byte        `json:"flags_raw"`
	Roll             byte        `json:"roll_raw"`
	Trip             int16       `json:"trip_index_raw"`
	Comment          string      `json:"comment"`
	Correction       int16       `json:"effective_declination_raw"`
	CorrectionSource string      `json:"declination_source"`
	Original         Values      `json:"original_values"`
	Values           Values      `json:"normalized_values_before_output_rounding"`
	EmittedStations  [2]string   `json:"emitted_from_to"`
	Emitted          [3]string   `json:"emitted_distance_azimuth_inclination"`
}
type Trip struct {
	Index       int         `json:"index"`
	Span        source.Span `json:"source_span"`
	Ticks       int64       `json:"recorded_unverified_ticks"`
	Declination int16       `json:"declination_raw"`
	Auto        bool        `json:"automatic"`
	Comment     string      `json:"comment"`
}
type Reference struct {
	Index    int         `json:"index"`
	Span     source.Span `json:"source_span"`
	Station  Identity    `json:"station"`
	East     int64       `json:"east_mm"`
	North    int64       `json:"north_mm"`
	Altitude int32       `json:"altitude_mm"`
	Comment  string      `json:"comment"`
}
type Values struct {
	Distance    float64 `json:"distance_metres"`
	Azimuth     float64 `json:"azimuth_degrees"`
	Inclination float64 `json:"inclination_degrees"`
}
type Group struct {
	Head      int                  `json:"head_index"`
	Kind      string               `json:"kind"`
	Members   []processing.Member  `json:"members"`
	Vector    processing.Vector    `json:"native_vector_mm"`
	Direction processing.Direction `json:"native_direction"`
}
type Directive struct {
	Line  int    `json:"svx_line"`
	Trip  int16  `json:"source_trip"`
	Value string `json:"value"`
}
type Loss struct {
	Code   string `json:"code"`
	Count  int    `json:"count"`
	Detail string `json:"detail"`
}
type Report struct {
	Schema        int             `json:"schema_version"`
	Tool          string          `json:"tool_version"`
	Format        string          `json:"format"`
	Context       string          `json:"context"`
	Profile       string          `json:"profile"`
	Policy        string          `json:"declination_policy"`
	SourceHash    string          `json:"source_sha256"`
	SourceSize    int             `json:"source_size_bytes"`
	SourceVersion byte            `json:"source_top_version"`
	Tail          source.TailKind `json:"source_tail_kind"`
	OutputHash    string          `json:"output_sha256"`
	OutputSize    int             `json:"output_size_bytes"`
	Stations      []Identity      `json:"stations"`
	Trips         []Trip          `json:"trips"`
	References    []Reference     `json:"omitted_references"`
	Measurements  []Measurement   `json:"measurements"`
	Groups        []Group         `json:"groups"`
	Dates         []Directive     `json:"commented_date_templates"`
	Declinations  []Directive     `json:"manual_declinations"`
	Losses        []Loss          `json:"losses"`
}
type Result struct {
	Text   []byte
	Report Report
}

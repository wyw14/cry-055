package buildinfo

// Source identifies the requirement record that produced this application.
type Source struct {
	ProjectTitle  string `json:"project_title"`
	RequirementID string `json:"requirement_id"`
	Sequence      int    `json:"sequence"`
	Category      string `json:"category"`
	Workbook      string `json:"workbook"`
	Sheet         string `json:"sheet"`
	Row           int    `json:"row"`
}

const (
	ProjectTitle   = "实验仪器校准与合格状态管理平台"
	RequirementID  = "GO-CG-050"
	GlobalSequence = 55
	SourceCategory = "代码生成"
	SourceWorkbook = "word2.xlsx"
	SourceSheet    = "Sheet1"
	SourceRow      = 491
)

// CurrentSource returns immutable provenance for diagnostics and support.
func CurrentSource() Source {
	return Source{
		ProjectTitle:  ProjectTitle,
		RequirementID: RequirementID,
		Sequence:      GlobalSequence,
		Category:      SourceCategory,
		Workbook:      SourceWorkbook,
		Sheet:         SourceSheet,
		Row:           SourceRow,
	}
}

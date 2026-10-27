package model

import "time"

// ID参数DTO
type IDDto struct {
	ID uint `json:"id" uri:"id"` // ID
}

type TaskTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`                                     // 任务名称
	Type      int       `gorm:"not null" json:"type"`                                              // 模板语言，见 TemplateType* 常量
	Content   string    `gorm:"type:text;not null" json:"content"`                                 // 任务内容
	Remark    string    `gorm:"size:500" json:"remark"`                                            // 备注信息
	CreatedBy string    `gorm:"size:50" json:"createdBy"`                                          // 创建人
	UpdatedBy string    `gorm:"size:50" json:"updatedBy"`                                          // 更新人
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt" time_format:"2006-01-02 15:04:05"` // 创建时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt" time_format:"2006-01-02 15:04:05"` // 更新时间
}

const (
	TemplateTypeShell      = 1
	TemplateTypePython     = 2
	TemplateTypeAnsible    = 3
	TemplateTypeJavaScript = 4
	TemplateTypePowerShell = 5
	TemplateTypeGo         = 6
	TemplateTypeRuby       = 7
	TemplateTypePHP        = 8
	TemplateTypePerl       = 9
	TemplateTypeCommand    = 10
)

// IsSupportedTemplateType 判断模板语言是否由执行器支持。
func IsSupportedTemplateType(templateType int) bool {
	return templateType >= TemplateTypeShell && templateType <= TemplateTypeCommand
}

func (TaskTemplate) TableName() string {
	return "task_template"
}

package domain

type Dependencies struct {
	Source    *Service  `json:"source" dynamodbav:"source"`
	DependsOn []Service `json:"depends_on" dyamodbav:"depends_on"`
}

package log

type BreakpointRuleType string

const (
	BreakpointRuleTypeIP      BreakpointRuleType = "ip"
	BreakpointRuleTypeUSER    BreakpointRuleType = "user"
	BreakpointRuleTypeGATEWAY BreakpointRuleType = "gateway"
	BreakpointRuleTypeHEADER  BreakpointRuleType = "header"
)

type BreakpointRuleItem struct {
	Type      BreakpointRuleType `json:"type" yaml:"type"`
	FieldName string             `json:"fieldName" yaml:"fieldName"`
	Data      string             `json:"data" yaml:"data"`
}

type BreakpointRule struct {
	Urls      []string             `json:"urls" yaml:"urls"`
	RuleTypes []BreakpointRuleItem `json:"ruleTypes" yaml:"ruleTypes"`
}

type BreakpointConfig struct {
	Enabled     bool             `json:"enabled" yaml:"enabled"`
	Rules       []BreakpointRule `json:"rules" yaml:"rules"`
	ServiceName string           `json:"serviceName" yaml:"serviceName"`
	Handler     func(dto BreakpointAddDto)
}

type BreakpointLogType string

const (
	BreakpointLogTypeRequest  BreakpointLogType = "request"
	BreakpointLogTypeResponse BreakpointLogType = "response"
)

type BreakpointAddDto struct {
	ApiUrl          string            `json:"apiUrl"`
	Body            string            `json:"body"`
	UrlParams       string            `json:"urlParams"`
	Headers         string            `json:"headers"`
	LogType         BreakpointLogType `json:"logType"`
	RequestType     string            `json:"requestType"`
	RequestId       string            `json:"requestId"`
	RequestIp       string            `json:"requestIp"`
	RequestTime     string            `json:"requestTime"`
	ServiceName     string            `json:"serviceName"`
	UserSession     string            `json:"userSession"`
	ResponseTime    string            `json:"responseTime"`
	ResponseHeaders string            `json:"responseHeaders"`
	HttpStatus      string            `json:"httpStatus"`
	Response        string            `json:"response"`
}

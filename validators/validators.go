package validators

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrTitleRequired = errors.New("标题不能为空")
	ErrTitleTooLong  = errors.New("标题不能超过200字符")
	ErrAmountInvalid = errors.New("金额范围无效")
	ErrDateInvalid   = errors.New("结束日期必须晚于开始日期")
	ErrCodeRequired  = errors.New("编码不能为空")
	ErrNameRequired  = errors.New("名称不能为空")
)

var (
	numberRegex = regexp.MustCompile(`^\d+$`)
	emailRegex  = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type ContractInput struct {
	Title      string     `json:"title"`
	Amount     float64    `json:"amount"`
	Currency   string     `json:"currency"`
	SignDate   *time.Time `json:"sign_date"`
	StartDate  *time.Time `json:"start_date"`
	EndDate    *time.Time `json:"end_date"`
	CustomerID uint       `json:"customer_id"`
	TypeID     uint       `json:"contract_type_id"`
}

func ValidateContract(input ContractInput) error {
	if strings.TrimSpace(input.Title) == "" {
		return ErrTitleRequired
	}
	if len([]rune(input.Title)) > 200 {
		return ErrTitleTooLong
	}
	if input.Amount < 0 {
		return ErrAmountInvalid
	}
	if input.Amount > 100000000 {
		return ErrAmountInvalid
	}
	if input.StartDate != nil && input.EndDate != nil {
		if input.EndDate.Before(*input.StartDate) {
			return ErrDateInvalid
		}
	}
	return nil
}

type CustomerInput struct {
	Name  string `json:"name"`
	Code  string `json:"code"`
	Type  string `json:"type"`
	Email string `json:"email"`
	Phone string `json:"contact_phone"`
}

func ValidateCustomer(input CustomerInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return ErrNameRequired
	}
	if len([]rune(input.Name)) > 200 {
		return errors.New("客户名称不能超过200字符")
	}
	if strings.TrimSpace(input.Code) == "" {
		return ErrCodeRequired
	}
	if input.Code != "" && len(input.Code) > 50 {
		return errors.New("客户编码不能超过50字符")
	}
	if input.Email != "" && !emailRegex.MatchString(input.Email) {
		return errors.New("邮箱格式无效")
	}
	return nil
}

type UserInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

func ValidateUser(input UserInput) error {
	if strings.TrimSpace(input.Username) == "" {
		return errors.New("用户名不能为空")
	}
	if len(input.Username) < 3 || len(input.Username) > 50 {
		return errors.New("用户名长度需在3-50字符之间")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(input.Username) {
		return errors.New("用户名只能包含字母、数字和下划线")
	}
	if input.Email != "" && !emailRegex.MatchString(input.Email) {
		return errors.New("邮箱格式无效")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 6 {
		return errors.New("密码至少6位")
	}
	return nil
}

type ReminderInput struct {
	ContractID   uint       `json:"contract_id"`
	ReminderDate *time.Time `json:"reminder_date"`
	DaysBefore   int        `json:"days_before"`
	Type         string     `json:"type"`
}

func ValidateReminder(input ReminderInput) error {
	if input.ContractID == 0 {
		return errors.New("合同ID不能为空")
	}
	if input.Type != "expiry" && input.Type != "payment" {
		return errors.New("提醒类型无效")
	}
	if input.DaysBefore < 1 || input.DaysBefore > 365 {
		return errors.New("提前天数需在1-365之间")
	}
	return nil
}

package admin

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// AdminTime 用于后台管理接口的时间格式化，输出 YYYY-MM-DD HH:mm:ss 格式。
type AdminTime time.Time

func (t AdminTime) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte("null"), nil
	}
	formatted := time.Time(t).Format("2006-01-02 15:04:05")
	return []byte(fmt.Sprintf(`"%s"`, formatted)), nil
}

func (t *AdminTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	parsed, err := time.Parse("2006-01-02 15:04:05", str)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, str)
		if err != nil {
			return err
		}
	}
	*t = AdminTime(parsed)
	return nil
}

func (t AdminTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

func (t *AdminTime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if v, ok := value.(time.Time); ok {
		*t = AdminTime(v)
		return nil
	}
	return fmt.Errorf("cannot scan type %T into AdminTime", value)
}

// ToAdminTime 将 *time.Time 转换为 *AdminTime
func ToAdminTime(t *time.Time) *AdminTime {
	if t == nil {
		return nil
	}
	adminT := AdminTime(*t)
	return &adminT
}

// ToAdminTimeRequired 将 time.Time 转换为 AdminTime
func ToAdminTimeRequired(t time.Time) AdminTime {
	return AdminTime(t)
}

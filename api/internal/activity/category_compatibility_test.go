package activity

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCategoryIDRequestSupportsLegacyAndNumericValues(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "数字", value: `4`, want: "4"},
		{name: "数字字符串", value: `"4"`, want: "4"},
		{name: "旧英文标识", value: `"sports"`, want: "sports"},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			var value categoryIDRequest
			if err := json.Unmarshal([]byte(item.value), &value); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if string(value) != item.want {
				t.Fatalf("category ID = %q, want %q", value, item.want)
			}
		})
	}
}

func TestAdaptCategoryIDResponseForLegacyApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/", nil)

	data := adaptCategoryIDResponse(context, PublicActivity{
		ID:         "100",
		CategoryID: 4,
	})
	body, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if result["categoryId"] != "4" {
		t.Fatalf("categoryId = %#v, want %q", result["categoryId"], "4")
	}

	categoryData := adaptCategoryIDResponse(context, []PublicActivityCategory{
		{ID: 4, Label: "运动"},
	})
	categoryBody, err := json.Marshal(categoryData)
	if err != nil {
		t.Fatalf("Marshal categories error = %v", err)
	}
	var categories []map[string]interface{}
	if err := json.Unmarshal(categoryBody, &categories); err != nil {
		t.Fatalf("Unmarshal categories error = %v", err)
	}
	if len(categories) != 1 || categories[0]["id"] != "4" {
		t.Fatalf("category id = %#v, want %q", categories, "4")
	}
}

func TestAdaptCategoryIDResponseForNumericApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(categoryIDVersionHeader, numericCategoryIDVersion)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	data := adaptCategoryIDResponse(context, PublicActivityCategory{
		ID:    4,
		Label: "运动",
	})
	category, ok := data.(PublicActivityCategory)
	if !ok {
		t.Fatalf("response type = %T, want PublicActivityCategory", data)
	}
	if category.ID != 4 {
		t.Fatalf("category ID = %d, want 4", category.ID)
	}
}

package uuid

import (
	"regexp"
	"testing"

	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestExec(t *testing.T) {
	t.Parallel()
	tool := New(nil, printer.GetDefault(), requester.NewFactory(), nil, nil)
	tests := []struct {
		name   string
		params *SystemUUIDParams
	}{
		{name: "nil params"},
		{name: "empty params", params: &SystemUUIDParams{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Exec(t.Context(), tt.params)
			if err != nil {
				t.Fatal(err)
			}
			response, ok := result.(*tools.Response)
			if !ok {
				t.Fatalf("Exec() result type = %T", result)
			}
			if response.Status != tools.StatusOK {
				t.Fatalf("Exec() status = %q", response.Status)
			}
			if !uuidV4Pattern.MatchString(response.Result) {
				t.Fatalf("Exec() result = %q, want UUID v4 format", response.Result)
			}
		})
	}
}

package violation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncident_GetFilePath(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "file URI with absolute path",
			uri:  "file:///path/to/file.java",
			want: "/path/to/file.java",
		},
		{
			name: "file URI with relative path",
			uri:  "file://relative/path.go",
			want: "relative/path.go",
		},
		{
			name: "plain path without file:// prefix",
			uri:  "/path/to/file.py",
			want: "/path/to/file.py",
		},
		{
			name: "relative path without file:// prefix",
			uri:  "src/main.go",
			want: "src/main.go",
		},
		{
			name: "empty URI",
			uri:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incident := &Incident{URI: tt.uri}
			got := incident.GetFilePath()
			assert.Equal(t, tt.want, got)
		})
	}
}

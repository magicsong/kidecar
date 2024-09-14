package hot_update

import "testing"

func Test_isValidVersion(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid version1",
			args: args{
				version: "v1.0.0",
			},
			want: true,
		},
		{
			name: "valid version2",
			args: args{
				version: "v1",
			},
			want: true,
		},
		{
			name: "invalid version2",
			args: args{
				version: "1.0",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidVersion(tt.args.version); got != tt.want {
				t.Errorf("isValidVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

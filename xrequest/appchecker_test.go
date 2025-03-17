package xrequest

import (
	"context"
	"testing"
)

type AppEnum string

const (
	AppEnum1 AppEnum = "test-app"
)

type TestRequest struct {
	App AppEnum
	ID  int
}

type NoAppRequest struct {
	ID int
}

func TestGetApp(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		req     interface{}
		want    string
		wantErr bool
	}{
		{
			name: "get app from context",
			ctx:  context.WithValue(context.Background(), "APP-ID", "test-app"),
			req:  &TestRequest{},
			want: "test-app",
		},
		{
			name: "get app from request struct",
			ctx:  context.Background(),
			req:  &TestRequest{App: "test-app"},
			want: "test-app",
		},
		{
			name:    "no app field in request",
			ctx:     context.Background(),
			req:     &NoAppRequest{ID: 1},
			want:    "",
			wantErr: true,
		},
		{
			name:    "non-struct request",
			ctx:     context.Background(),
			req:     "invalid",
			want:    "",
			wantErr: false,
		},
		{
			name: "nil request",
			ctx:  context.Background(),
			req:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetApp(tt.ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetApp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetApp() = %v, want %v", got, tt.want)
			}
		})
	}
}

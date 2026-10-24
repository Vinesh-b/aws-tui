package services

import (
	"slices"
	"testing"

	"aws-tui/internal/ui/uitest"
)

func newServicesView() *ServicesHomeView {
	var view = NewServicesHomeView(uitest.NewAppContext())
	view.AddItem("Lambda", "Lambdas and logs", '1', func() {})
	view.AddItem("DynamoDB", "View and search tables", '2', func() {})
	view.AddItem("S3", "Buckets and objects", '3', func() {})
	return view
}

func listedServices(view *ServicesHomeView) []string {
	var names []string
	for i := range view.filteredList.GetItemCount() {
		var mainText, _ = view.filteredList.GetItemText(i)
		names = append(names, mainText)
	}
	return names
}

func TestServicesHomeView__ListsAddedServicesInOrder(t *testing.T) {
	var view = newServicesView()

	if got := listedServices(view); !slices.Equal(got, []string{"Lambda", "DynamoDB", "S3"}) {
		t.Errorf("services = %v", got)
	}
}

func TestServicesHomeView__SearchFiltersByNameAndDescription(t *testing.T) {
	var tests = []struct {
		search string
		want   []string
	}{
		{"dynamo", []string{"DynamoDB"}},
		{"buckets", []string{"S3"}},
		{"nothing-matches", nil},
	}

	for _, tt := range tests {
		t.Run(tt.search, func(t *testing.T) {
			var view = newServicesView()

			view.SetSearchText(tt.search)

			if got := listedServices(view); !slices.Equal(got, tt.want) {
				t.Errorf("services = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServicesHomeView__ClearingSearchRestoresAllServices(t *testing.T) {
	var view = newServicesView()
	view.SetSearchText("dynamo")

	view.SetSearchText("")

	if got := listedServices(view); !slices.Equal(got, []string{"Lambda", "DynamoDB", "S3"}) {
		t.Errorf("services = %v", got)
	}
}

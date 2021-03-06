package albumbot

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// https://github.com/awsdocs/aws-doc-sdk-examples/blob/main/gov2/dynamodb/ScanItems/ScanItemsv2_test.go
// TODO: ちゃんとテストする
// たぶん github.com/golang/mock/gomock を使う？
type ScanAPIClientImpl struct{}

func (sc ScanAPIClientImpl) Scan(
	_ context.Context,
	_ *dynamodb.ScanInput,
	_ ...func(*dynamodb.Options),
) (*dynamodb.ScanOutput, error) {
	item1 := Album{Title: "test1"}
	item2 := Album{Title: "test2"}

	av1, err := attributevalue.MarshalMap(item1)
	if err != nil {
		return nil, errors.New("Could not items")
	}

	av2, err := attributevalue.MarshalMap(item2)
	if err != nil {
		return nil, errors.New("Could not items")
	}

	avs := []map[string]types.AttributeValue{
		av1,
		av2,
	}
	return &dynamodb.ScanOutput{Items: avs}, nil
}
func TestGetAlbumTitles(t *testing.T) {
	mockClient := &ScanAPIClientImpl{}
	tests := []struct {
		name       string
		wantTitles []string
		wantErr    bool
	}{{
		name:       "test",
		wantTitles: []string{"test1", "test2"},
		wantErr:    false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitles, err := getAlbumTitles(tableForTest, context.Background(), mockClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAlbumTitles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTitles, tt.wantTitles) {
				t.Errorf("GetAlbumTitles() = %v, want %v", gotTitles, tt.wantTitles)
			}
		})
	}
}
func TestGetAlbumUrls(t *testing.T) {
	wants := []string{
		"https://test1.png",
		"https://test2.png",
		"https://test3.png",
	}
	type args struct {
		table string
		title string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{{
		name: "_test",
		args: args{
			table: tableForTest,
			title: "_test",
		},
		want: wants,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAlbumUrls(tt.args.table, tt.args.title)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAlbumUrls() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAlbumPage(t *testing.T) {
	normalWants := []string{
		"https://test1.png",
		"https://test2.png",
	}
	overCountWants := []string{
		"https://test2.png",
		"https://test3.png",
	}
	type args struct {
		title string
		start int
		count int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "normal case",
			args: args{
				title: "_test",
				start: 0,
				count: 2,
			},
			want: normalWants,
		},
		{
			name: "over count case",
			args: args{
				title: "_test",
				start: 1,
				count: 9999,
			},
			want: overCountWants,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := GetAlbumPage(tableForTest, tt.args.title, tt.args.start, tt.args.count); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAlbumPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostImage(t *testing.T) {
	type args struct {
		albumTitle string
		url        string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "error",
			args: args{
				albumTitle: "invisible taisho",
				url:        "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := PostImage(tableForTest, tt.args.albumTitle, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("PostImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateAlbum(t *testing.T) {
	type args struct {
		table string
		title string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "error",
			args: args{
				table: tableForTest,
				title: "_test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateAlbum(tt.args.table, tt.args.title); (err != nil) != tt.wantErr {
				t.Errorf("CreateAlbum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateAndDeleteAlbum(t *testing.T) {
	type args struct {
		table string
		title string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal case",
			args: args{
				table: tableForTest,
				title: "_testForCreateAndDeleteAlbum",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateAlbum(tt.args.table, tt.args.title); (err != nil) != tt.wantErr {
				t.Errorf("CreateAlbum() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := DeleteAlbum(tt.args.table, tt.args.title); (err != nil) != tt.wantErr {
				t.Errorf("DeleteAlbum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChangeAlbumTitle(t *testing.T) {
	type args struct {
		table string
		old   string
		new   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal case",
			args: args{
				table: tableForTest,
				old:   "_testOld",
				new:   "_testNew",
			},
			wantErr: false,
		},
		{
			name: "error case",
			args: args{
				table: tableForTest,
				old:   "_testNew",
				new:   "_test",
			},
			wantErr: true,
		},
		{
			name: "post process",
			args: args{
				table: tableForTest,
				old:   "_testNew",
				new:   "_testOld",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ChangeAlbumTitle(tt.args.table, tt.args.old, tt.args.new); (err != nil) != tt.wantErr {
				t.Errorf("ChangeAlbumTitle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPostAndDeleteImage(t *testing.T) {
	type args struct {
		table string
		title string
		url   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal case",
			args: args{
				table: tableForTest,
				title: "_test",
				url:   "testForPostAndDeleteImage",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := PostImage(tt.args.table, tt.args.title, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("PostImage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := DeleteImage(tt.args.table, tt.args.title, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("DeleteImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

const tableForTest = "Albums"

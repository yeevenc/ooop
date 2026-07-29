package activity

import (
	"errors"
	"testing"
)

func TestCreateInputLimitsActivityImages(t *testing.T) {
	input := validCreateInputForImageTest()
	input.ImageURLs = []string{
		"https://source.example.com/1.jpg",
		"https://source.example.com/2.jpg",
		"https://source.example.com/3.jpg",
		"https://source.example.com/4.jpg",
		"https://source.example.com/5.jpg",
		"https://source.example.com/6.jpg",
	}

	if _, err := input.toModel(3000); !errors.Is(err, ErrTooManyImages) {
		t.Fatalf("toModel() error = %v, want ErrTooManyImages", err)
	}
}

func TestCreateInputAllowsFiveActivityImages(t *testing.T) {
	input := validCreateInputForImageTest()
	input.ImageURLs = []string{
		"https://source.example.com/1.jpg",
		"https://source.example.com/2.jpg",
		"https://source.example.com/3.jpg",
		"https://source.example.com/4.jpg",
		"https://source.example.com/5.jpg",
	}

	item, err := input.toModel(3000)
	if err != nil {
		t.Fatalf("toModel() error = %v", err)
	}
	if gallery := decodeGallery(item.GalleryJSON, item.ImageURL); len(gallery) != 5 {
		t.Fatalf("gallery length = %d, want 5", len(gallery))
	}
}

func validCreateInputForImageTest() CreateInput {
	return CreateInput{
		Title:        "周末羽毛球",
		CategoryID:   4,
		LocationText: "羽毛球馆",
		City:         "西安市",
		Latitude:     34.2,
		Longitude:    108.9,
		TotalCount:   10,
		Intro:        "一起打球",
	}
}

package models

type ImageType string

const (
	ProfileImage ImageType = "profile"
	ItemImage    ImageType = "item"
	OfferImage ImageType = "offer"
)

var BucketLink = "yoursdaily-3e32c.appspot.com"

type Image struct {
	Bucket string `json:"-" db:"bucket"`
	Path   string `json:"-" db:"path"`
}

func IsValidImageType(imageType string) bool {
	return imageType == string(OfferImage) || imageType == string(ItemImage)
}
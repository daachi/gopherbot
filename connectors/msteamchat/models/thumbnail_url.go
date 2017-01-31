package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
)

// ThumbnailURL Object describing a media thumbnail
// swagger:model ThumbnailUrl
type ThumbnailURL struct {

	// Alt text to display for screen readers on the thumbnail image
	Alt string `json:"alt,omitempty"`

	// url pointing to an thumbnail to use for media content
	URL string `json:"url,omitempty"`
}

// Validate validates this thumbnail Url
func (m *ThumbnailURL) Validate(formats strfmt.Registry) error {
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
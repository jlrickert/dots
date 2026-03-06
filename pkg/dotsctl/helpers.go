package dotsctl

import (
	"errors"

	"github.com/jlrickert/dots/pkg/dots"
)

func isNotExist(err error) bool {
	return errors.Is(err, dots.ErrNotExist)
}

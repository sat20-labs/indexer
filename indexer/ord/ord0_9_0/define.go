package ord0_9_0

import (
	"errors"
)

var (
	ErrInvalidInscription = errors.New("invalid inscription structure or field")
	ErrNoInscription      = errors.New("no inscription found at current position")
	ErrNoTapscript        = errors.New("witness does not contain a tapscript")
)

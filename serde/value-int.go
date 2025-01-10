package serde

type InvalidValue struct{}

var _ SourceValue = InvalidValue{}

func (i InvalidValue) Bool() (bool, error) {
	return false, ErrInvalidType
}

func (i InvalidValue) Int() (int64, error) {
	return 0, ErrInvalidType
}

func (i InvalidValue) Float() (float64, error) {
	return 0, ErrInvalidType
}

func (i InvalidValue) String() (string, error) {
	return "", ErrInvalidType
}

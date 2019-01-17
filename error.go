package scope

import "fmt"

type errorlist []error

func (e *errorlist) append(err error) {
	if err != nil {
		*e = append(*e, err)
	}
}

func (e errorlist) err() error {
	if len(e) == 0 {
		return nil
	}
	return e
}

func (e errorlist) Error() string {
	switch len(e) {
	case 0:
		return "no error"
	case 1:
		return e[0].Error()
	default:
		return fmt.Sprintf("%v (and %d more errors)", e[0], len(e)-1)
	}
}

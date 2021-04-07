package errhelp

func SecondErr(_ interface{}, err error) error {
	return err
}

func MustString(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

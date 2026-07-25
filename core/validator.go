package core

func ValidateDocument(d *Document) error {
	if d == nil {
		return nil
	}
	return d.Validate()
}

package db

// Save persists the schema to local cache
func (s *Schema) Save() error {
	return SaveSchema(s)
}

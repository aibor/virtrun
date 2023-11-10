package files

// Type defines the type of an [Entry].
type Type int

const (
	// A regular file is copied completely into the archive.
	TypeRegular Type = iota
	// A directory is created in the archive. Parent directories are not created
	// automatically. Ensure to create the complete file tree yourself.
	TypeDirectory
	// A symbolic link in the archive.
	TypeLink
)

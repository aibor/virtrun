package initramfs

// FileType defines the type of a [TreeNode].
type FileType int

const (
	// A regular file is copied completely into the archive.
	FileTypeRegular FileType = iota
	// A directory is created in the archive. Parent directories are not created
	// automatically. Ensure to create the complete file tree yourself.
	FileTypeDirectory
	// A symbolic link in the archive.
	FileTypeLink
	// A file with its content written from an io.Reader instead of being
	// copied from the fs.
	FileTypeVirtual
)

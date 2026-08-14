package stitcher

// CompiledDocument represents the final assembled master markdown document.
type CompiledDocument struct {
	Title        string
	Description  string
	Content      []byte
	FileCount    int
	HeadingCount int
	ByteCount    int
}

// TOCItem represents an entry in the Table of Contents.
type TOCItem struct {
	Title  string
	Level  int // 1..6
	Anchor string
}

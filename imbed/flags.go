package imbed

import "bytes"

type ImbedFlag int

const (
	// Compress assets whenever possible
	CompressAssets ImbedFlag = 1 << iota

	// Build FileSystem API
	BuildFsAPI

	// Build union FileSystem API (implies BuildFsAPI)
	BuildUnionFsAPI

	// Build http.FileSystem API (implies BuildFsAPI)
	BuildHttpFsAPI

	// Build http.Server handler API
	BuildHttpHandlerAPI

	// Build raw data access API (dangerous)
	BuildRawBytesAPI

	// Build main function (implies BuildHttpHandlerAPI and BuildFsAPI)
	BuildMain

	maxFlag uint = iota
)

func (f ImbedFlag) has(flag ImbedFlag) bool {
	return (f&flag)!=0
}

func (f ImbedFlag) CompressAssets() bool      { return f.has(CompressAssets) }
func (f ImbedFlag) BuildFsAPI() bool          { return f.has(BuildFsAPI) }
func (f ImbedFlag) BuildUnionFsAPI() bool     { return f.has(BuildUnionFsAPI) }
func (f ImbedFlag) BuildHttpFsAPI() bool      { return f.has(BuildHttpFsAPI) }
func (f ImbedFlag) BuildHttpHandlerAPI() bool { return f.has(BuildHttpHandlerAPI) }
func (f ImbedFlag) BuildRawBytesAPI() bool    { return f.has(BuildRawBytesAPI) }
func (f ImbedFlag) BuildMain() bool           { return f.has(BuildMain) }

func (f ImbedFlag) Set(s ImbedFlag, c bool) ImbedFlag {
	if c {
		return f | s
	} else {
		return f
	}
}

func (f ImbedFlag) String() string {
	var buf bytes.Buffer
	var add = func(s string) {
		if buf.Len() > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(s)
	}
	if f.has(CompressAssets) {
		add("Compress")
	}
	if f.has(BuildHttpHandlerAPI) {
		add("Http")
	}
	if f.has(BuildFsAPI) {
		add("FS")
	}
	if f.has(BuildHttpFsAPI) {
		add("HttpFS")
	}
	if f.has(BuildUnionFsAPI) {
		add("UnionFS")
	}
	if f.has(BuildMain) {
		add("Main")
	}
	return buf.String()
}


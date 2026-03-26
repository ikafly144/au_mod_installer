package musmgr

type parsedFile struct {
	Path           string
	URLs           []string
	Type           string
	ExtractPath    *string
	TargetPlatform string
}

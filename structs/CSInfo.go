package structs

type FileCSInfo struct {
	BlockIndex int64 // index of block in file
	CS16       string
	CS64       string
	CS128      string
}

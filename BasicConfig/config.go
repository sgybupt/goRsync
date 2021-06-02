package BasicConfig

var LocalPathPrefix string
var LocalRPCPort int
var PortRangeL int
var PortRangeH int

func init() {
	LocalPathPrefix = "/Users/su/ftp_test/server"
	PortRangeL = 10000
	PortRangeH = 20000
}

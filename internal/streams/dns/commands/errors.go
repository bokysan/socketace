package commands

// Declare a few of standard error responses
var (
	BadVersion    ClientServerResponse = "BADVER"
	BadLen        ClientServerResponse = "BADLEN"
	BadIp         ClientServerResponse = "BADIP"
	BadCodec      ClientServerResponse = "BADCODEC"
	BadFrag       ClientServerResponse = "BADFRAG"
	BadServerFull ClientServerResponse = "VFUL"
	VersionOk     ClientServerResponse = "VACK"
	LazyModeOk    ClientServerResponse = "LACK"
	VersionNotOk  ClientServerResponse = "VNAK"
)

var BadErrors = []ClientServerResponse{
	BadVersion, BadLen, BadIp, BadCodec, BadFrag, BadServerFull,
}

type ClientServerResponse string

func (cse *ClientServerResponse) Error() string {
	return string(*cse)
}

// Is will check if the first few bytes of the suppied byte array match the given exception.
func (cse *ClientServerResponse) Is(data []byte) bool {
	return len(data) >= len(*cse) && string(data[:len(*cse)]) == string(*cse)
}

// Strip will remove the prefix of the ClientServerResponse from the given stream. No validation takes place.
func (cse *ClientServerResponse) Strip(data []byte) []byte {
	return data[len(*cse):]
}

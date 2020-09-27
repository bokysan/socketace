package packet

const (
	// 64kb + Max MTU size + 1
	BufferSize = 64*1024 + 1518 + 1
)

//goland:noinspection SpellCheckingInspection
var Magic = []byte("FACABABE59C3677D625C4900A20730DF2A5C27BE")
var MagicLength = len(Magic)

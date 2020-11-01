package dns

import "golang.org/x/net/dns/dnsmessage"

type QueryTypes []dnsmessage.Type

// Define different query types which can be used
const (
	QueryTypeNull    = dnsmessage.Type(10)
	QueryTypePrivate = dnsmessage.Type(65000)
	QueryTypeTxt     = dnsmessage.TypeTXT
	QueryTypeSrv     = dnsmessage.TypeSRV
	QueryTypeMx      = dnsmessage.TypeMX
	QueryTypeCname   = dnsmessage.TypeCNAME
	QueryTypeAAAA    = dnsmessage.TypeAAAA
	QueryTypeA       = dnsmessage.TypeA
)

// Sorted by priority
var QueryTypesByPriority = QueryTypes{
	QueryTypeNull,
	QueryTypePrivate,
	QueryTypeTxt,
	QueryTypeSrv,
	QueryTypeMx,
	QueryTypeCname,
	QueryTypeAAAA,
	QueryTypeA,
}

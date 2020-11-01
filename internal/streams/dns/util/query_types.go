package util

import "golang.org/x/net/dns/dnsmessage"

type QueryTypes []dnsmessage.Type

// Define different query types which can be used
var (
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

// Before returns true if q1 has higher priority than q2
func (qt QueryTypes) Before(q1, q2 dnsmessage.Type) bool {
	var i1, i2 int
	for i, q := range qt {
		if q == q1 {
			i1 = i
		} else if q == q2 {
			i2 = i
		}
	}
	return i1 < i2
}

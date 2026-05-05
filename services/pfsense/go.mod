module github.com/jlfowle/asterism/services/pfsense

go 1.24.0

require github.com/gosnmp/gosnmp v1.43.2

require github.com/jlfowle/asterism/pkg/authz v0.0.0-00010101000000-000000000000 // indirect

replace github.com/jlfowle/asterism/pkg/authz => ../../pkg/authz

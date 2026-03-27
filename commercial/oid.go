package commercial

import "encoding/asn1"

// OID prefix: 1.3.6.1.4.1.15099.1
// 15099 is Yao's internal port number, used as a recognizable enterprise number
// for this closed-loop certificate system. Not registered with IANA.

var (
	// Scope
	OIDProduct = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 1, 1}
	OIDEdition = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 1, 2}
	OIDEnv     = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 1, 3}
	OIDDomain  = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 1, 4}
	OIDAppID   = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 1, 5}

	// Quota
	OIDMaxUsers     = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 2, 1}
	OIDMaxTaiNodes  = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 2, 2}
	OIDMaxAgents    = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 2, 3}
	OIDMaxSandboxes = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 2, 4}
	OIDMaxAPIRPM    = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 2, 5}
	OIDMaxStorageGB = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 2, 6}

	// Permissions
	OIDAllowBrandingRemoval = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 3, 1}
	OIDAllowWhiteLabel      = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 3, 2}
	OIDAllowMultiTenant     = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 3, 3}
	OIDAllowCustomDomain    = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 3, 4}
	OIDAllowHostExec        = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 3, 5}
	OIDAllowSSO             = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 3, 6}
	OIDSupportLevel         = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 3, 7}

	// Binding (optional — if present, must match at runtime)
	OIDMachineID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 5, 1}

	// Issuance (internal tracking)
	OIDIssuerID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 4, 1}
	OIDOrderID  = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 4, 2}
	OIDNote     = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 15099, 1, 4, 3}
)
